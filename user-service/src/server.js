/**
 * Server Entry Point
 * This file starts the Express server.
 * Separated from app.js to allow Supertest to import the app without starting the server.
 */
require('dotenv').config();
const app = require('./app');

const PORT = process.env.PORT || 3001;

app.listen(PORT, () => {
    console.log(`🚀 User Service running on port ${PORT}`);
});
