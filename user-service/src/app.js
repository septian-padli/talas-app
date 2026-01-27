require('dotenv').config();
const express = require('express');
const cors = require('cors');
const cookieParser = require('cookie-parser');

const app = express();
const PORT = process.env.PORT || 3001;

// 1. Middlewares
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(cookieParser());
app.use(cors({
    origin: 'http://localhost:3000', // URL Frontend nanti
    credentials: true // Izinkan Cookie lewat
}));

// 2. Routing
const authRoutes = require('./routes/authRoutes');
app.use('/api/auth', authRoutes);

const userRoutes = require('./routes/userRoutes');
app.use('/api/users', userRoutes);


const internalRoutes = require('./routes/internalRoutes');
app.use('/api/internal', internalRoutes); // Protected by Service Secret

// Routing Test (Langsung tembak dulu untuk tes Nginx)
// Nginx mengirim request ke /api/auth/test, jadi kita tangkap path yang sama
app.get('/api/auth/init-testing', (req, res) => {
    res.json({
        status: 'success',
        message: 'Halo! User Service berhasil terhubung dengan Nginx Gateway!',
        service: 'User Service',
        timestamp: new Date().toISOString()
    });
});

// 3. Start Server
app.listen(PORT, () => {
    console.log(`🚀 User Service running on port ${PORT}`);
});