const { PrismaClient } = require('@prisma/client');

/**
 * Prisma Client Singleton
 * Mencegah error "Too many connections" saat development (Hot Reload).
 */
const prisma = global.prisma || new PrismaClient();

if (process.env.NODE_ENV !== 'production') {
  global.prisma = prisma;
}

module.exports = prisma;
