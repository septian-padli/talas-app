import axiosInstance from "@/lib/axios"; // Sesuaikan import axios kamu
// Removed unused import CreateShowcaseRequest
// import { CreateShowcaseRequest } from "@/types/showcase";

export const showcaseService = {
	// POST Create Showcase
	createShowcase: async (payload: FormData) => {
		// Accepts FormData or object
		let config = {};
		if (payload instanceof FormData) {
			config = { headers: { "Content-Type": "multipart/form-data" } };
		}
		const response = await axiosInstance.post("/showcases", payload, config);
		return response.data;
	},

	// POST Upload Media (Asumsi endpoint upload terpisah)
	uploadMedia: async (formData: FormData) => {
		const response = await axiosInstance.post(
			"/showcases/media/upload",
			formData,
			{
				headers: { "Content-Type": "multipart/form-data" },
			},
		);
		return response.data.data;
	},
};
