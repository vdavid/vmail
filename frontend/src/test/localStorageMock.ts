// This file must be imported FIRST, before any MSW imports
// MSW's CookieStore tries to access localStorage during module initialization

const createLocalStorageMock = () => {
    const store = new Map<string, string>()
    // noinspection JSUnusedGlobalSymbols
    return {
        getItem: (key: string) => store.get(key) ?? null,
        setItem: (key: string, value: string) => {
            store.set(key, value)
        },
        removeItem: (key: string) => {
            store.delete(key)
        },
        clear: () => {
            store.clear()
        },
        get length() {
            return store.size
        },
        key: (index: number) => {
            const keys = Array.from(store.keys())
            return keys[index] ?? null
        },
    }
}

const localStorageMock = createLocalStorageMock()

// Set up localStorage on all possible global objects
const setLocalStorage = (target: typeof globalThis | Window) => {
    try {
        Object.defineProperty(target, 'localStorage', {
            value: localStorageMock,
            writable: true,
            configurable: true,
            enumerable: true,
        })
    } catch {
        // Ignore errors
    }
}

// Set up on all possible global scopes
if (typeof globalThis !== 'undefined') {
    setLocalStorage(globalThis)
}
if (typeof window !== 'undefined') {
    setLocalStorage(window)
}

// Use type assertion for global objects that may have localStorage
interface GlobalWithLocalStorage {
    localStorage?: Storage
}
;(globalThis as GlobalWithLocalStorage).localStorage = localStorageMock
if (typeof window !== 'undefined') {
    ;(window as GlobalWithLocalStorage).localStorage = localStorageMock
}
