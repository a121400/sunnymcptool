import './global-shim.js';
import Java from 'frida-java-bridge';

function installHooks() {
    Java.perform(function () {
        var AppBrandCommonBindingJni = Java.use(
            'com.tencent.mm.appbrand.commonjni.AppBrandCommonBindingJni'
        );

        AppBrandCommonBindingJni['nativeInvokeHandler']
            .overload('java.lang.String', 'java.lang.String', 'java.lang.String', 'int', 'boolean', 'int', 'int')
            .implementation = function (jsapi_name, data, str3, counter, z16, i17, i18) {
                send(JSON.stringify({
                    type: 'cloud-request',
                    apiType: jsapi_name,
                    id: counter,
                    timestamp: Date.now(),
                    param1: jsapi_name,
                    param2: data
                }));
                return this['nativeInvokeHandler'](jsapi_name, data, str3, counter, z16, i17, i18);
            };

        AppBrandCommonBindingJni['invokeCallbackHandler']
            .overload('int', 'java.lang.String', 'java.lang.String')
            .implementation = function (counter, res, extra) {
                send(JSON.stringify({
                    type: 'cloud-response',
                    id: counter,
                    timestamp: Date.now(),
                    param1: res,
                    param2: extra
                }));
                return this['invokeCallbackHandler'](counter, res, extra);
            };

        AppBrandCommonBindingJni['invokeCallbackHandler']
            .overload('int', 'java.lang.String')
            .implementation = function (counter, res) {
                send(JSON.stringify({
                    type: 'cloud-response',
                    id: counter,
                    timestamp: Date.now(),
                    param1: res
                }));
                return this['invokeCallbackHandler'](counter, res);
            };

        var AppBrandJsBridgeBinding = Java.use(
            'com.tencent.mm.appbrand.commonjni.AppBrandJsBridgeBinding'
        );

        AppBrandJsBridgeBinding['invokeCallbackHandler']
            .overload('int', 'java.lang.String', 'java.lang.String')
            .implementation = function (counter, res, extra) {
                send(JSON.stringify({
                    type: 'cloud-response',
                    apiType: 'worker',
                    id: counter,
                    timestamp: Date.now(),
                    param1: res,
                    param2: extra
                }));
                return this['invokeCallbackHandler'](counter, res, extra);
            };

        AppBrandJsBridgeBinding['invokeCallbackHandler']
            .overload('int', 'java.lang.String')
            .implementation = function (counter, res) {
                send(JSON.stringify({
                    type: 'cloud-response',
                    apiType: 'worker',
                    id: counter,
                    timestamp: Date.now(),
                    param1: res
                }));
                return this['invokeCallbackHandler'](counter, res);
            };
    });
}

send(JSON.stringify({
    type: 'log',
    message: '[Android hook] Java.available=' + Java.available
}));

if (Java.available) {
    try {
        installHooks();
        send(JSON.stringify({ type: 'hook-ready', platform: 'android', timestamp: Date.now() }));
    } catch (e) {
        send(JSON.stringify({ type: 'error', message: 'install error: ' + e + ' / ' + (e.stack || '') }));
    }
} else {
    send(JSON.stringify({ type: 'error', message: 'Java not available' }));
}
