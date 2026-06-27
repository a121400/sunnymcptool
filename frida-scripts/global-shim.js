if (typeof global === 'undefined') {
    Object.defineProperty(globalThis, 'global', { value: globalThis, writable: true, configurable: true });
}
export default globalThis;
