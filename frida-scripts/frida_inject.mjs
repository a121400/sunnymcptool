import frida from "frida";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const scriptPath = path.join(__dirname, "cloudHookAndroid.js");

function log(msg) {
  process.stdout.write(JSON.stringify({ type: "send", payload: JSON.stringify({ type: "log", message: msg }) }) + "\n");
}

async function main() {
  const device = await frida.getUsbDevice({ timeout: 10000 });
  const processes = await device.enumerateProcesses();
  
  const wechatProcs = processes.filter(p => 
    p.name.includes("com.tencent.mm") || p.name === "WeChat"
  );

  if (wechatProcs.length === 0) {
    log("未找到微信进程");
    process.exit(1);
  }

  log(`找到 ${wechatProcs.length} 个微信进程`);
  wechatProcs.forEach(p => log(`  进程: ${p.name} (PID:${p.pid})`));

  const compiler = new frida.Compiler();
  const bundle = await compiler.build(scriptPath, { projectRoot: __dirname });

  let hookedCount = 0;
  for (const proc of wechatProcs) {
    try {
      const session = await device.attach(proc.pid);
      const script = await session.createScript(bundle);

      script.message.connect((message) => {
        if (message.type === "send") {
          process.stdout.write(JSON.stringify({ type: "send", payload: message.payload }) + "\n");
        } else if (message.type === "error") {
          process.stdout.write(JSON.stringify({ type: "error", payload: `[${proc.name}] ${message.description}` }) + "\n");
        }
      });

      session.detached.connect((reason) => {
        log(`${proc.name}(${proc.pid}) 断开: ${reason}`);
      });

      await script.load();
      hookedCount++;
      log(`Hook: ${proc.name} (PID:${proc.pid})`);
    } catch (e) {
      log(`失败 ${proc.name}(${proc.pid}): ${e.message}`);
    }
  }

  if (hookedCount === 0) {
    log("所有进程 hook 均失败");
    process.exit(1);
  }

  log(`hook 完成: ${hookedCount}/${wechatProcs.length}`);
}

main().catch(err => {
  process.stdout.write(JSON.stringify({ type: "error", payload: err.message }) + "\n");
  process.exit(1);
});
