import { create } from 'zustand'

interface UIState {
    selectedThreadIndex: number | null
    setSelectedThreadIndex: (index: number | null) => void
    incrementSelectedIndex: (maxIndex: number) => void
    decrementSelectedIndex: () => void
}

export const useUIStore = create<UIState>((set) => ({
    selectedThreadIndex: null,
    setSelectedThreadIndex: (index: number | null) => {
        set({ selectedThreadIndex: index })
    },
    incrementSelectedIndex: (maxIndex: number) => {
        set((state) => ({
            selectedThreadIndex:
                state.selectedThreadIndex === null
                    ? 0
                    : Math.min(state.selectedThreadIndex + 1, maxIndex - 1),
        }))
    },
    decrementSelectedIndex: () => {
        set((state) => ({
            selectedThreadIndex:
                state.selectedThreadIndex === null || state.selectedThreadIndex === 0
                    ? null
                    : state.selectedThreadIndex - 1,
        }))
    },
}))
