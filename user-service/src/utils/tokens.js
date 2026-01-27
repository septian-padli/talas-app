const jwt = require('jsonwebtoken');

const JWT_SECRET = process.env.JWT_SECRET || 'secret';
const ACCESS_TOKEN_EXPIRY = '15m'; // 15 Menit
const REFRESH_TOKEN_EXPIRY = '7d'; // 7 Hari

/**
 * Generate Access Token (Short-lived)
 * Payload: { id, username, email }
 */
const generateAccessToken = (user) => {
  return jwt.sign(
    {
      id: user.id,
      username: user.username,
      email: user.email
    },
    JWT_SECRET,
    { expiresIn: ACCESS_TOKEN_EXPIRY }
  );
};

/**
 * Generate Refresh Token (Long-lived)
 * Payload: { id } - Minimalist
 */
const generateRefreshToken = (user) => {
  return jwt.sign(
    { id: user.id },
    JWT_SECRET,
    { expiresIn: REFRESH_TOKEN_EXPIRY }
  );
};

/**
 * Verify Token
 */
const verifyToken = (token) => {
  try {
    return jwt.verify(token, JWT_SECRET);
  } catch (error) {
    return null;
  }
};

module.exports = {
  generateAccessToken,
  generateRefreshToken,
  verifyToken
};
