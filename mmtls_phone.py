# -*- coding: utf-8 -*-
"""Sunny 启动后改手机 mmtls 密钥，并把业务包明文交给 Sunny 显示。"""
import hashlib
import os
import subprocess
import tempfile
import threading
import time

import frida

# 无窗口运行：本脚本由 pythonw 拉起，自己再 spawn adb 时也要显式禁掉控制台，
# 否则每次 pidof 都会闪一个黑框。
CREATE_NO_WINDOW = 0x08000000 if os.name == "nt" else 0


def run_quiet(args, timeout=8):
    return subprocess.run(
        args,
        capture_output=True,
        text=True,
        timeout=timeout,
        creationflags=CREATE_NO_WINDOW,
    )

KEY_FILE = os.path.join(os.path.expanduser("~"), "Sunny", "mmtls_key.txt")
CANDIDATES = os.path.join(os.path.expanduser("~"), "Sunny", "mmtls_candidates.txt")
BIZ_FILE = os.path.join(os.path.expanduser("~"), "Sunny", "mmtls_biz.bin")
API_FILE = os.path.join(os.path.expanduser("~"), "Sunny", "mmtls_biz_api.bin")

PSK_JS = r"""
var pskHex = "%s";
var psk = [];
for (var i = 0; i < pskHex.length; i += 2) {
  psk.push(parseInt(pskHex.substr(i, 2), 16));
}
var n = 0;
function writePsk(p) {
  try { p.writeByteArray(psk); return true; } catch (e) { return false; }
}
function noteKey(where, p) {
  try {
    var hex = Array.from(new Uint8Array(p.readByteArray(32))).map(function (b) {
      return ("0" + b.toString(16)).slice(-2);
    }).join("");
    send({ t: "livepsk", where: where, key: hex });
  } catch (e) {}
}
var net = Process.findModuleByName("libwechatnetwork.so");
if (net) {
  Interceptor.attach(net.base.add(0x224e9c), {
    onEnter: function (args) {
      try {
        var obj = args[1];
        if (obj.add(16).readU32() !== 32) return;
        noteKey("early", obj.add(8).readPointer());
      } catch (e) {}
    }
  });
}
var crypto = null;
Process.enumerateModules().forEach(function (m) {
  if (m.name === "libcrypto.so" && m.path.indexOf("/com.tencent.mm-") >= 0) crypto = m;
});
if (crypto) {
  var init = crypto.findExportByName("HMAC_Init_ex");
  if (init) {
    Interceptor.attach(init, {
      onEnter: function (args) {
        var m = Process.findModuleByAddress(this.context.lr);
        if (!m || m.name.indexOf("wechatnetwork") < 0) return;
        if (args[2].toInt32() !== 32) return;
        noteKey("hmac", args[1]);
      }
    });
  }
}
send({ t: "ready", net: net ? net.base.toString() : "missing" });
setInterval(function () { send({ t: "tick", n: n }); }, 5000);
"""

BIZ_JS = r"""
var mm = Process.findModuleByName("libMMProtocalJni.so");
var n = 0;
function hasText(p, len) {
  try {
    var a = new Uint8Array(p.readByteArray(Math.min(len, 500)));
    var run = 0;
    for (var i = 0; i < a.length; i++) {
      if (a[i] >= 32 && a[i] < 127) {
        run++;
        if (run >= 8) return true;
      } else run = 0;
    }
  } catch (e) {}
  return false;
}
if (mm) {
  Interceptor.attach(mm.base.add(0x778e0), {
    onEnter: function () {
      var len = this.context.x2.toInt32();
      if (len < 64 || len > 300000) return;
      var p = ptr(this.context.x1);
      if (!hasText(p, len)) return;
      n++;
      if (n <= 4) send({ t: "biz", n: n, len: len });
      send({ t: "bizbody", len: len }, p.readByteArray(len));
    }
  });
}
send({ t: "bizready", has: !!mm });
setInterval(function () { send({ t: "biztick", n: n }); }, 5000);
"""


def log(line):
    try:
        print(line, flush=True)
    except Exception:
        pass


def pidof(name):
    try:
        r = run_quiet(["adb", "shell", "pidof", name])
    except Exception as e:
        log("adb " + str(e))
        return 0
    text = (r.stdout or "").strip().split()
    if text and text[0].isdigit():
        return int(text[0])
    return 0


def load_psk():
    if not os.path.isfile(KEY_FILE):
        return ""
    for line in open(KEY_FILE, encoding="utf-8", errors="replace"):
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        name, val = line.split("=", 1)
        if name.strip() == "psk":
            val = val.strip()
            if len(val) == 64:
                return val
    return ""


def key_stamp():
    try:
        return os.stat(KEY_FILE).st_mtime
    except Exception:
        return 0


def write_atomic(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    for _ in range(5):
        fd, tmp = tempfile.mkstemp(dir=os.path.dirname(path), prefix=".biz")
        try:
            os.write(fd, data)
            os.close(fd)
            fd = None
            os.replace(tmp, path)
            return
        except Exception as e:
            log("write " + path + " " + str(e))
            time.sleep(0.05)
        finally:
            if fd is not None:
                try:
                    os.close(fd)
                except Exception:
                    pass
            if os.path.isfile(tmp):
                try:
                    os.remove(tmp)
                except Exception:
                    pass


def is_api(data):
    return (
        b"tcbapi_call_gateway" in data
        or b"qbase_commapi" in data
        or b"participate" in data
        or b"/draw/" in data
    )


def save_biz(data):
    if not data or len(data) < 64:
        return
    write_atomic(BIZ_FILE, data)
    if is_api(data):
        write_atomic(API_FILE, data)
        log("api %s" % len(data))


def hold(pid, js, on_message):
    dev = frida.get_usb_device(timeout=5)
    session = dev.attach(pid)
    dead = {"v": False}

    def on_detached(reason, crash):
        dead["v"] = True
        log("detached %s %s" % (pid, reason))

    session.on("detached", on_detached)
    script = session.create_script(js)
    script.on("message", on_message)
    script.load()
    return session, dead


def append_candidate(hexkey):
    hexkey = (hexkey or "").strip().lower()
    if len(hexkey) != 64:
        return
    lines = []
    if os.path.isfile(CANDIDATES):
        lines = [ln.strip() for ln in open(CANDIDATES, encoding="utf-8", errors="replace") if len(ln.strip()) == 64]
    if hexkey in lines:
        return
    lines.append(hexkey)
    if len(lines) > 40:
        lines = lines[-40:]
    os.makedirs(os.path.dirname(CANDIDATES), exist_ok=True)
    with open(CANDIDATES, "w", encoding="utf-8", newline="\n") as f:
        f.write("\n".join(lines) + "\n")
    log("candidate " + hexkey[:16])


def psk_loop():
    while True:
        if len(load_psk()) != 64:
            time.sleep(2)
            continue
        pid = pidof("com.tencent.mm:push") or pidof("com.tencent.mm")
        if pid <= 0:
            time.sleep(2)
            continue
        state = {"stamp": key_stamp()}

        def on_message(msg, data, state=state):
            payload = msg.get("payload") or msg
            if isinstance(payload, dict) and payload.get("t") == "livepsk":
                append_candidate(payload.get("key") or "")
                return
            if isinstance(payload, dict) and payload.get("t") == "tick":
                return
            log(str(payload))

        try:
            session, dead = hold(pid, PSK_JS % load_psk(), on_message)
            log("psk attached %s" % pid)
            while not dead["v"]:
                time.sleep(1)
                now = pidof("com.tencent.mm:push") or pidof("com.tencent.mm")
                if key_stamp() != state["stamp"] or now != pid:
                    break
            try:
                session.detach()
            except Exception:
                pass
        except Exception as e:
            log("psk fail " + str(e))
            time.sleep(2)


def biz_loop():
    last = {"h": b""}
    while True:
        pid = pidof("com.tencent.mm")
        push = pidof("com.tencent.mm:push")
        if pid <= 0 or pid == push:
            time.sleep(2)
            continue

        def on_message(msg, data):
            payload = msg.get("payload") or msg
            if data:
                h = hashlib.sha256(data).digest()
                if h != last["h"]:
                    last["h"] = h
                    save_biz(data)
                    log("biz %s" % len(data))
                else:
                    try:
                        os.utime(BIZ_FILE, None)
                    except Exception:
                        pass
                return
            if isinstance(payload, dict) and payload.get("t") == "biztick":
                return
            log(str(payload))

        try:
            session, dead = hold(pid, BIZ_JS, on_message)
            log("biz attached %s" % pid)
            while not dead["v"]:
                time.sleep(1)
                if pidof("com.tencent.mm") != pid:
                    break
            try:
                session.detach()
            except Exception:
                pass
        except Exception as e:
            log("biz fail " + str(e))
            time.sleep(2)


def main():
    log("phone sync start")
    threading.Thread(target=biz_loop, daemon=True).start()
    psk_loop()


if __name__ == "__main__":
    main()
