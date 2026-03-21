// src/lib/axios.ts
import axios from "axios";
const publicPaths = ["/about", "/login", "/register"];

const api = axios.create({
	baseURL: process.env.NEXT_PUBLIC_API_URL,
	headers: {
		"Content-Type": "application/json",
	},
	// 🔥 PENTING: Wajib true agar cookie dikirim/diterima
	withCredentials: true,
});

// --- LOGIC ANTRIAN (QUEUE) ---
// Variabel ini di luar interceptor agar statusnya global
let isRefreshing = false;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let failedQueue: any[] = [];

// Fungsi untuk memproses antrian setelah refresh selesai
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const processQueue = (error: any, token: string | null = null) => {
	failedQueue.forEach((prom) => {
		if (error) {
			prom.reject(error);
		} else {
			prom.resolve(token);
		}
	});
	failedQueue = [];
};

api.interceptors.response.use(
	(response) => response,
	async (error) => {
		const originalRequest = error.config;

		// 1. Cek Error 401
		// PENTING: Pastikan yang error BUKAN request ke /auth/refresh itu sendiri
		// Jika /auth/refresh error 401, berarti memang sesi habis total.
		if (
			error.response?.status === 401 &&
			!originalRequest.url.includes("/auth/refresh")
		) {
			// Cek pencegah loop infinite
			if (originalRequest._retry) {
				return Promise.reject(error);
			}

			// 2. Jika sedang ada proses refresh berlangsung...
			if (isRefreshing) {
				// ...Request ini kita suruh ANTRI (masuk failedQueue)
				// Dia akan menunggu sampai processQueue dipanggil
				return new Promise(function (resolve, reject) {
					failedQueue.push({ resolve, reject });
				})
					.then(() => {
						// Setelah antrian jalan, ulangi request ini
						return api(originalRequest);
					})
					.catch((err) => {
						return Promise.reject(err);
					});
			}

			// 3. Jika belum ada yang refresh, request ini yang jadi EKSEKUTOR
			originalRequest._retry = true;
			isRefreshing = true;

			try {
				// Tembak Refresh Token
				await api.post("/auth/refresh");

				// SUKSES: Proses semua request yang mengantri tadi
				processQueue(null, "success");

				// Ulangi request EKSEKUTOR ini
				return api(originalRequest);
			} catch (refreshError) {
				// GAGAL: Beritahu semua antrian bahwa refresh gagal
				processQueue(refreshError, null);

				// Tidak ada redirect paksa ke login. Biarkan error diproses oleh handler di UI.
				return Promise.reject(refreshError);
			} finally {
				// Reset status agar siap untuk refresh berikutnya di masa depan
				isRefreshing = false;
			}
		}

		return Promise.reject(error);
	},
);

export default api;
