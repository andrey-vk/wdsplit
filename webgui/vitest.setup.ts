// Node 22+ ships an experimental native localStorage/sessionStorage that
// can shadow jsdom's own implementation (getItem/setItem missing
// entirely), depending on how Node was invoked — version-dependent and
// not something to chase with CLI flags. Install a real, self-contained
// Storage implementation directly instead.
class MemoryStorage implements Storage {
  private store = new Map<string, string>()

  get length() {
    return this.store.size
  }

  clear() {
    this.store.clear()
  }

  getItem(key: string) {
    return this.store.has(key) ? this.store.get(key)! : null
  }

  key(index: number) {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string) {
    this.store.delete(key)
  }

  setItem(key: string, value: string) {
    this.store.set(key, String(value))
  }
}

for (const target of [globalThis, window]) {
  Object.defineProperty(target, 'localStorage', {
    value: new MemoryStorage(),
    configurable: true,
  })
}

// jsdom doesn't implement matchMedia at all — a well-known, unrelated gap.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList
}
