import axiosInstance from "@/lib/axios"; // Sesuaikan import axios kamu
import { CreateShowcaseRequest, Category } from "@/types/showcase";

export const showcaseService = {
	// GET Categories untuk dropdown
	getCategories: async () => {
		const response = await axiosInstance.get<{ data: Category[] }>(
			"/categories",
		);
		return response.data; // Sesuaikan dengan wrapper response API kamu
	},

	// POST Create Showcase
	createShowcase: async (payload: CreateShowcaseRequest) => {
		const response = await axiosInstance.post("/showcases", payload);
		return response.data;
	},

	// POST Upload Media (Asumsi endpoint upload terpisah)
	uploadMedia: async (formData: FormData) => {
		const response = await axiosInstance.post("/media/upload", formData, {
			headers: { "Content-Type": "multipart/form-data" },
		});
		return response.data.data; // Return { url: "...", type: "..." }
	},
};
