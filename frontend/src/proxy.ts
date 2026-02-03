// src/proxy.ts
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function proxy(request: NextRequest) {
	// 1. Ambil cookie token dari request browser
	// ⚠️ PENTING: Pastikan nama string 'token' ini SAMA dengan key yang dikirim Backend
	// (Cek di Inspect Element -> Application -> Cookies jika ragu)
	const token = request.cookies.get("accessToken")?.value;
	const refreshToken = request.cookies.get("refreshToken")?.value;

	// 2. Cek user sedang ada di halaman mana
	const { pathname } = request.nextUrl;

	// Daftar halaman Auth (Login/Register)
	const isAuthPage =
		pathname.startsWith("/login") || pathname.startsWith("/register");

	// Daftar halaman Public (jika ada halaman landing page yang boleh diakses tanpa login)
	// const isPublicPage = pathname === '/about' || pathname === '/privacy';

	// --- LOGIC PENGAMANAN ---

	// SKENARIO A: User SUDAH Login (Punya Token)
	// Tapi mencoba masuk ke halaman Login/Register
	// Action: Tendang balik ke Dashboard (Home)
	if (token && isAuthPage) {
		// if (refreshToken && (pathname.startsWith("/login") || pathname.startsWith("/register"))) {
		return NextResponse.redirect(new URL("/", request.url));
	}

	// SKENARIO B: User BELUM Login (Tidak punya Token)
	// Tapi mencoba masuk ke halaman selain Auth (misal: Dashboard)
	// Action: Tendang ke halaman Login
	if (!token && !refreshToken && !isAuthPage) {
		return NextResponse.redirect(new URL("/login", request.url));
	}

	// Jika tidak masuk skenario di atas, biarkan lewat
	return NextResponse.next();
}

// Konfigurasi Matcher:
// Proxy ini akan jalan di SEMUA route, KECUALI:
// - /api (biarkan backend handle auth api)
// - /_next/static (file statis nextjs)
// - /_next/image (gambar nextjs)
// - favicon.ico
// - gambar-gambar publik (svg, png, jpg)
export const config = {
	matcher: [
		"/((?!api|_next/static|_next/image|favicon.ico|.*\\.svg|.*\\.png|.*\\.jpg).*)",
	],
};
