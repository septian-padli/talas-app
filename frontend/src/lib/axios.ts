// src/lib/axios.ts
import axios from "axios";

const api = axios.create({
	baseURL: process.env.NEXT_PUBLIC_API_URL,
	headers: {
		"Content-Type": "application/json",
	},
	// 🔥 PENTING: Ini kuncinya agar cookie dikirim/diterima
	withCredentials: true,
});

// HAPUS interceptor request yang menyisipkan 'Bearer token' manual.
// Kita hanya butuh response interceptor untuk handle 401 (Logout).

api.interceptors.response.use(
	(response) => response,
	async (error) => {
		const originalRequest = error.config;

		// Cek jika error 401 DAN belum pernah mencoba refresh sebelumnya
		if (error.response?.status === 401 && !originalRequest._retry) {
			originalRequest._retry = true; // Tandai agar tidak looping infinite

			try {
				// 1. Coba minta Access Token baru ke Backend
				// Pastikan endpoint ini ada di backend kamu!
				await api.post("/auth/refresh");

				// 2. Jika berhasil, ulangi request awal yang tadi gagal
				return api(originalRequest);
			} catch (refreshError) {
				// 3. Jika refresh token juga sudah basi (misal user offline 30 hari)
				// BARU kita logout paksa
				if (typeof window !== "undefined") {
					localStorage.removeItem("user_data");
					window.location.href = "/login";
				}
				return Promise.reject(refreshError);
			}
		}

		return Promise.reject(error);
	},
);
export default api;
