// src/hooks/useUser.ts
import { useQuery } from "@tanstack/react-query";
import { authService } from "@/services/authService";

export const useUser = () => {
	return useQuery({
		queryKey: ["user"], // Kunci unik untuk cache
		queryFn: authService.getMe,

		// Konfigurasi Penting:
		retry: false, // Jika gagal (401), jangan coba lagi (anggap guest)
		staleTime: 1000 * 60 * 5, // Data dianggap "fresh" selama 5 menit
		refetchOnWindowFocus: false, // Jangan refresh saat pindah tab (kecuali perlu real-time)
	});
};
