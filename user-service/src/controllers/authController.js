const prisma = require('../utils/prisma');
const { hashPassword } = require('../utils/password');
const { registerSchema } = require('../validations/authValidation');

/**
 * Register User Baru
 * POST /api/auth/register
 */
const register = async (req, res) => {
  try {
    // 1. Validasi Input (Zod)
    const validation = registerSchema.safeParse(req.body);
    if (!validation.success) {
      const errorDetails = validation.error.issues.map(err => ({
        field: err.path[0],
        message: err.message
      }));
      
      return res.status(400).json({
        code: 400,
        success: false,
        message: 'Validasi gagal',
        errors: errorDetails
      });
    }

    const { username, email, password, name } = validation.data;

    // 2. Cek Duplikasi (Email / Username)
    const existingUser = await prisma.user.findFirst({
      where: {
        OR: [
          { email: email },
          { username: username }
        ]
      }
    });

    if (existingUser) {
      // Tentukan field mana yang duplikat untuk pesan error yang spesifik
      const field = existingUser.email === email ? 'email' : 'username';
      return res.status(400).json({
        code: 400,
        success: false,
        message: 'User sudah terdaftar',
        errors: [{
          field: field,
          message: `${field} sudah digunakan`
        }]
      });
    }

    // 3. Hash Password
    const hashedPassword = await hashPassword(password);

    // 4. Create User
    const newUser = await prisma.user.create({
      data: {
        username,
        email,
        password: hashedPassword,
        name
      },
      select: {
        id: true,
        email: true,
        username: true,
        name: true,
        createdAt: true
      }
    });

    // 5. Response Sukses (Sesuai API Contract)
    res.status(201).json({
      code: 201,
      success: true,
      message: 'User registered successfully',
      data: {
        user: newUser
      }
    });

  } catch (error) {
    console.error('Register Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

module.exports = {
  register
};

const { loginSchema } = require('../validations/authValidation');
const { comparePassword } = require('../utils/password');
const { generateAccessToken, generateRefreshToken, verifyToken } = require('../utils/tokens');

// ----------------------------------------------------
// LOGIN
// ----------------------------------------------------
const login = async (req, res) => {
  try {
    // 1. Validasi Input
    const validation = loginSchema.safeParse(req.body);
    if (!validation.success) {
      const errorDetails = validation.error.issues.map(err => ({
        field: err.path[0],
        message: err.message
      }));

      return res.status(400).json({
        code: 400,
        success: false,
        message: 'Validasi gagal',
        errors: errorDetails
      });
    }

    const { email, password } = validation.data;

    // 2. Cari User by Email
    const user = await prisma.user.findUnique({ where: { email } });
    if (!user) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Email atau password salah',
        errors: null
      });
    }

    // 3. Cek Password
    const isPasswordValid = await comparePassword(password, user.password);
    if (!isPasswordValid) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Email atau password salah',
        errors: null
      });
    }

    // 4. Generate Tokens
    const accessToken = generateAccessToken(user);
    const refreshToken = generateRefreshToken(user);

    // 5. Simpan Refresh Token ke DB (Whitelist)
    // Hitung tanggal kedaluwarsa (7 hari dari sekarang)
    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 7);

    await prisma.refreshToken.create({
      data: {
        token: refreshToken,
        userId: user.id,
        expiresAt: expiresAt
      }
    });

    // 6. Set Cookies (HttpOnly)
    // Access Token: 15 menit
    res.cookie('accessToken', accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      maxAge: 15 * 60 * 1000 // 15 min
    });

    // Refresh Token: 7 hari
    res.cookie('refreshToken', refreshToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      maxAge: 7 * 24 * 60 * 60 * 1000 // 7 days
    });

    // 7. Response JSON (Sesuai API Contract)
    res.json({
      code: 200,
      success: true,
      message: 'Login berhasil',
      data: {
        user: {
          id: user.id,
          username: user.username,
          email: user.email,
          name: user.name,
          avatar_url: user.avatarUrl
        }
      }
    });

  } catch (error) {
    console.error('Login Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

// ----------------------------------------------------
// REFRESH TOKEN
// ----------------------------------------------------
const refresh = async (req, res) => {
  try {
    // 1. Ambil Refresh Token dari Cookie
    const refreshToken = req.cookies.refreshToken;
    if (!refreshToken) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Unauthorized: No Refresh Token',
        errors: null
      });
    }

    // 2. Verify Token (Signature check)
    const decoded = verifyToken(refreshToken);
    if (!decoded) {
      return res.status(403).json({
        code: 403,
        success: false,
        message: 'Forbidden: Invalid Refresh Token',
        errors: null
      });
    }

    // 3. Cek Whitelist di Database
    const savedToken = await prisma.refreshToken.findUnique({
      where: { token: refreshToken },
      include: { user: true } // Ambil sekalian data usernya
    });

    if (!savedToken) {
      return res.status(403).json({
        code: 403,
        success: false,
        message: 'Forbidden: Token Reused or Revoked',
        errors: null
      });
    }

    // 4. Cek Expiry Date (DB)
    if (new Date() > savedToken.expiresAt) {
      // Hapus token expired
      await prisma.refreshToken.delete({ where: { id: savedToken.id } });
      return res.status(403).json({
        code: 403,
        success: false,
        message: 'Forbidden: Refresh Token Expired',
        errors: null
      });
    }

    // 5. Generate NEW Access Token
    const newAccessToken = generateAccessToken(savedToken.user);

    // 6. Update Cookie Access Token
    res.cookie('accessToken', newAccessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      maxAge: 15 * 60 * 1000 // 15 min
    });

    res.json({
      code: 200,
      success: true,
      message: 'Token refreshed',
      data: null
    });

  } catch (error) {
    console.error('Refresh Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

// ----------------------------------------------------
// LOGOUT
// ----------------------------------------------------
const getMe = async (req, res) => {
  try {
    // Data user sudah di-attach oleh middleware protect
    const user = req.user;
    
    res.json({
      code: 200,
      success: true,
      message: 'Berhasil mengambil data user',
      data: {
        user: user
      }
    });
  } catch (error) {
    console.error('GetMe Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

// ----------------------------------------------------
// LOGOUT
// ----------------------------------------------------
const logout = async (req, res) => {
  try {
    const refreshToken = req.cookies.refreshToken;
    
    // Note: Request ini sudah diprotek middleware, jadi req.user ada.
    // Tapi kita tetap fokus menghapus token yang dikirim client.
    
    if (refreshToken) {
      await prisma.refreshToken.delete({
        where: { token: refreshToken }
      }).catch(() => {});
    }

    // Clear Cookies dengan opsi yang SAMA saat login (Penting!)
    const cookieOptions = {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production'
    };

    res.clearCookie('accessToken', cookieOptions);
    res.clearCookie('refreshToken', cookieOptions);

    res.json({
      code: 200,
      success: true,
      message: 'Logout berhasil',
      data: null
    });

  } catch (error) {
    console.error('Logout Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

module.exports = {
  register,
  login,
  refresh,
  logout,
  getMe
};
