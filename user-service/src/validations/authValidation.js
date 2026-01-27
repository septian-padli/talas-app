const { z } = require('zod');

/**
 * Schema Validasi Register
 * Rules:
 * - username: alphanumeric, min 3
 * - email: valid email
 * - password: min 6
 * - name: min 2
 */
const registerSchema = z.object({
  username: z.string()
    .min(3, 'Username minimal 3 karakter')
    .regex(/^[a-zA-Z0-9_]+$/, 'Username hanya boleh huruf, angka, dan underscore'),
  
  email: z.string()
    .email('Format email tidak valid'),
  
  password: z.string()
    .min(6, 'Password minimal 6 karakter'),
  
  name: z.string()
    .min(2, 'Nama minimal 2 karakter')
});

/**
 * Schema Validasi Login
 */
const loginSchema = z.object({
  email: z.string().email('Format email tidak valid'),
  password: z.string().min(1, 'Password harus diisi')
});

module.exports = {
  registerSchema,
  loginSchema
};
