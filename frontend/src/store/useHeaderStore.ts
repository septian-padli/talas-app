import { create } from "zustand";

interface HeaderState {
	title: string;
	setTitle: (newTitle: string) => void;
}

export const useHeaderStore = create<HeaderState>((set) => ({
	title: "Talas App", // Default title
	setTitle: (newTitle) => set({ title: newTitle }),
}));
