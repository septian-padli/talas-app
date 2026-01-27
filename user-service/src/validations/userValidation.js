const { z } = require('zod');

/**
 * Enum SocialPlatform must match Prisma Schema
 */
const SocialPlatform = z.enum([
  'INSTAGRAM',
  'FACEBOOK',
  'GITHUB',
  'X',
  'LINKEDIN',
  'DRIBBBLE'
]);

/**
 * Schema Validasi Update Profile
 * PATCH /api/users/me
 */
const updateProfileSchema = z.object({
  name: z.string().min(2, 'Nama minimal 2 karakter').optional(),
  bio: z.string().max(300, 'Bio maksimal 300 karakter').optional(),
  jobTitle: z.string().max(100, 'Job Title maksimal 100 karakter').optional(),
  avatarUrl: z.string().url('Avatar URL tidak valid').optional(),
  
  socialLinks: z.array(
    z.object({
      social: SocialPlatform,
      link: z.string().url('Link media sosial harus berupa URL valid'),
      username: z.string().optional()
    })
  ).optional()
});

module.exports = {
  updateProfileSchema
};
