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
	(error) => {
		if (error.response && error.response.status === 401) {
			if (typeof window !== "undefined") {
				// Hapus data user di UI (bukan token, karena token di cookie)
				localStorage.removeItem("user_data");
				// Redirect login
				// window.location.href = '/login';
			}
		}
		return Promise.reject(error);
	},
);

export default api;
