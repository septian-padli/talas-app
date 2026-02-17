import axiosInstance from "@/lib/axios"; // Sesuaikan import axios kamu
import { CategoryListResponse } from "@/types/showcase";

export const categoryService = {
	// GET Categories untuk dropdown
	getCategories: async () => {
		const response =
			await axiosInstance.get<CategoryListResponse>("/categories");
		return response.data;
	},
};
