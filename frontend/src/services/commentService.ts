import axiosInstance from "@/lib/axios";

export const commentService = {
	// POST Create Comment
	createComment: async (showcaseId: string, content: string) => {
		const response = await axiosInstance.post(
			`/showcases/${showcaseId}/comments`,
			{ content },
		);
		return response.data;
	},
};
