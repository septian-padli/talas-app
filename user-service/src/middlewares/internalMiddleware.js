const verifyInternalKey = (req, res, next) => {
  const clientSecret = req.headers['x-service-secret'];
  const serverSecret = process.env.INTERNAL_SERVICE_SECRET;

  if (!serverSecret) {
    console.error('INTERNAL_SERVICE_SECRET is not defined in environment variables');
    return res.status(500).json({
      code: 500,
      success: false,
      message: 'Internal Server Error: Security configuration missing'
    });
  }

  if (!clientSecret || clientSecret !== serverSecret) {
    return res.status(403).json({
      code: 403,
      success: false,
      message: 'Forbidden: Invalid Service Secret'
    });
  }

  next();
};

module.exports = {
  verifyInternalKey
};
