// src/lib/axios.ts
import axios from "axios";

// 1. Buat Instance
const api = axios.create({
	baseURL: process.env.NEXT_PUBLIC_API_URL,
	headers: {
		"Content-Type": "application/json",
	},
});

// 2. Request Interceptor: Sisipkan Token
api.interceptors.request.use(
	(config) => {
		// Cek apakah kode jalan di browser (bukan server side Next.js)
		if (typeof window !== "undefined") {
			const token = localStorage.getItem("token");
			if (token) {
				config.headers.Authorization = `Bearer ${token}`;
			}
		}
		return config;
	},
	(error) => Promise.reject(error),
);

// 3. Response Interceptor: Handle Token Expired (401)
api.interceptors.response.use(
	(response) => response,
	(error) => {
		// Jika error 401 (Unauthorized), berarti token basi/salah
		if (error.response && error.response.status === 401) {
			if (typeof window !== "undefined") {
				localStorage.removeItem("token"); // Hapus token
				// Opsional: Redirect ke halaman login
				// window.location.href = '/login';
			}
		}
		return Promise.reject(error);
	},
);

export default api;
