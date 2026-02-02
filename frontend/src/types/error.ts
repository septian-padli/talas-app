export interface FieldError {
	field: string;
	message: string;
}

export interface ApiErrorResponse {
	code: number;
	success: boolean;
	message: string;
	errors?: FieldError[];
}
