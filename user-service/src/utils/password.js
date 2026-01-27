const bcrypt = require('bcryptjs');

/**
 * Hash password menggunakan bcrypt
 * @param {string} password - Password raw
 * @returns {Promise<string>} - Password yang sudah di-hash
 */
const hashPassword = async (password) => {
  const salt = await bcrypt.genSalt(10);
  return await bcrypt.hash(password, salt);
};

/**
 * Bandingkan password raw dengan hash
 * @param {string} password - Password raw
 * @param {string} hashedPassword - Password hash dari DB
 * @returns {Promise<boolean>} - True jika cocok
 */
const comparePassword = async (password, hashedPassword) => {
  return await bcrypt.compare(password, hashedPassword);
};

module.exports = {
  hashPassword,
  comparePassword
};
