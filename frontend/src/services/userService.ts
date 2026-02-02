import axios from "@/lib/axios";
import type {
	UpdateProfilePayload,
	GetMeResponse,
	UpdateProfileResponse,
	GetPublicProfileResponse,
	ToggleFollowResponse,
	GetFollowersResponse,
	GetFollowingResponse,
} from "@/types/user";

export const userService = {
	async getMe(): Promise<GetMeResponse> {
		const res = await axios.get("/users/me");
		return res.data;
	},

	async updateProfile(
		data: UpdateProfilePayload,
	): Promise<UpdateProfileResponse> {
		const res = await axios.patch("/users/me", data);
		return res.data;
	},

	async getByUsername(username: string): Promise<GetPublicProfileResponse> {
		const res = await axios.get(`/users/${username}`);
		return res.data;
	},

	async toggleFollow(userId: string): Promise<ToggleFollowResponse> {
		const res = await axios.post(`/users/${userId}/follow`);
		return res.data;
	},

	async getFollowers(userId: string): Promise<GetFollowersResponse> {
		const res = await axios.get(`/users/${userId}/followers`);
		return res.data;
	},

	async getFollowing(userId: string): Promise<GetFollowingResponse> {
		const res = await axios.get(`/users/${userId}/following`);
		return res.data;
	},
};
