// Register user baru
export interface RegisterRequest {
	email: string;
	password: string;
	username: string;
	name?: string;
}

// Login user
export interface LoginRequest {
	email: string;
	password: string;
}

// =====================
// Response Interfaces
// =====================

// /auth/register, /auth/login, /auth/refresh
export interface AuthResponse {
	code: number;
	success: boolean;
	message: string;
	data?: {
		user?: UserProfile;
		// Token dikirim via Set-Cookie, bukan di body
	};
}

// /auth/logout
export interface LogoutResponse {
	code: number;
	success: boolean;
	message: string;
}

// /auth/me (get current logged-in user)
export interface MeResponse {
	code: number;
	success: boolean;
	data: UserProfile;
}

// UserProfile sesuai schema di API_CONTRACT
export interface UserProfile {
	id: string;
	name: string;
	username: string;
	email: string;
	avatar_url: string | null;
	bio: string | null;
	is_verified: boolean;
	created_at: string;
	updated_at: string;
}
