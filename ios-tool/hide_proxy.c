#include <dlfcn.h>
#include <stddef.h>

typedef void *CFDictionaryRef;
typedef void *CFAllocatorRef;
typedef const void *CFDictionaryKeyCallBacks_t;
typedef const void *CFDictionaryValueCallBacks_t;
typedef void (*MSHookFunction_t)(void *symbol, void *replacement, void **original);
typedef CFDictionaryRef (*CFDictionaryCreate_t)(CFAllocatorRef, const void **, const void **, long, CFDictionaryKeyCallBacks_t, CFDictionaryValueCallBacks_t);

static void *(*orig_proxy_func)(void);
static CFDictionaryCreate_t real_CFDictionaryCreate;
static CFDictionaryKeyCallBacks_t real_kCFTypeDictionaryKeyCallBacks;
static CFDictionaryValueCallBacks_t real_kCFTypeDictionaryValueCallBacks;

static void *hook_proxy_func(void) {
    if (real_CFDictionaryCreate && real_kCFTypeDictionaryKeyCallBacks && real_kCFTypeDictionaryValueCallBacks) {
        return real_CFDictionaryCreate(NULL, NULL, NULL, 0,
            real_kCFTypeDictionaryKeyCallBacks, real_kCFTypeDictionaryValueCallBacks);
    }
    return orig_proxy_func ? orig_proxy_func() : NULL;
}

__attribute__((constructor))
static void init(void) {
    void *substrate = dlopen("/var/jb/usr/lib/libsubstrate.dylib", RTLD_NOW);
    if (!substrate) substrate = dlopen("/usr/lib/libsubstrate.dylib", RTLD_NOW);
    if (!substrate) return;

    MSHookFunction_t hook = (MSHookFunction_t)dlsym(substrate, "MSHookFunction");
    if (!hook) return;

    void *cf = dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", RTLD_NOW);
    if (cf) {
        real_CFDictionaryCreate = (CFDictionaryCreate_t)dlsym(cf, "CFDictionaryCreate");
        real_kCFTypeDictionaryKeyCallBacks = (CFDictionaryKeyCallBacks_t)dlsym(cf, "kCFTypeDictionaryKeyCallBacks");
        real_kCFTypeDictionaryValueCallBacks = (CFDictionaryValueCallBacks_t)dlsym(cf, "kCFTypeDictionaryValueCallBacks");
    }

    void *cfnetwork = dlopen("/System/Library/Frameworks/CFNetwork.framework/CFNetwork", RTLD_NOW);
    if (!cfnetwork) return;

    void *sym = dlsym(cfnetwork, "CFNetworkCopySystemProxySettings");
    if (!sym) return;

    hook(sym, (void *)hook_proxy_func, (void **)&orig_proxy_func);
}
