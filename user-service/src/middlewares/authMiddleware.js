const prisma = require('../utils/prisma');
const { verifyToken } = require('../utils/tokens');

/**
 * Middleware Proteksi Endpoint
 * @param path protected
 * 1. Cek Cookie accessToken
 * 2. Verify Token
 * 3. Attach User to req.user
 */
const protect = async (req, res, next) => {
  try {
    // 1. Ambil token dari Cookie (Bukan Header)
    const token = req.cookies.accessToken;

    if (!token) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Unauthorized: Harap login terlebih dahulu',
        errors: null
      });
    }

    // 2. Verify Token
    const decoded = verifyToken(token);
    if (!decoded) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Unauthorized: Token tidak valid atau expired',
        errors: null
      });
    }

    // 3. (Optional) Cek User di Database
    // Exclude password demi keamanan
    const user = await prisma.user.findUnique({
      where: { id: decoded.id },
      select: {
        id: true,
        username: true,
        email: true,
        name: true,
        avatarUrl: true,
        jobTitle: true
      }
    });

    if (!user) {
      return res.status(401).json({
        code: 401,
        success: false,
        message: 'Unauthorized: User tidak ditemukan',
        errors: null
      });
    }

    // 4. Attach user ke request
    req.user = user;
    next();

  } catch (error) {
    console.error('Auth Middleware Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal pada auth middleware',
      errors: null
    });
  }
};

module.exports = { protect };
