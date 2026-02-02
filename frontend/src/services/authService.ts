// src/services/authService.ts
import api from "@/lib/axios";
import { LoginRequest, RegisterRequest } from "@/types/auth";

export const authService = {
	login: async (data: LoginRequest) => {
		// Browser otomatis menangkap header 'Set-Cookie' dari respon ini
		const response = await api.post("/auth/login", data);
		return response.data;
	},

	register: async (data: RegisterRequest) => {
		const response = await api.post("/auth/register", data);
		return response.data;
	},

	logout: async () => {
		// PENTING: Kita harus minta Backend menghapus cookienya
		// Pastikan backend punya endpoint POST /auth/logout yang return 'Set-Cookie: token=; Max-Age=0'
		try {
			await api.post("/auth/logout");
		} catch (err) {
			console.error("Logout error", err);
		} finally {
			// Bersihkan state lokal UI
			if (typeof window !== "undefined") {
				localStorage.removeItem("user_data");
				window.location.href = "/login";
			}
		}
	},

	getMe: async () => {
		// Endpoint ini membaca HttpOnly cookie secara otomatis
		const response = await api.get("/auth/me");
		return response.data; // Asumsi return object User
	},
};
