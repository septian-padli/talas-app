const { PrismaClient } = require('@prisma/client');
const { faker } = require('@faker-js/faker');
const bcrypt = require('bcryptjs');

const prisma = new PrismaClient();

async function main() {
  console.log('🌱 Starting Seeder...');

  // 1. Cleanup Database
  await prisma.notification.deleteMany();
  await prisma.socialLink.deleteMany();
  await prisma.follow.deleteMany();
  await prisma.refreshToken.deleteMany();
  await prisma.user.deleteMany();
  
  console.log('🧹 Database Cleaned Up');

  // 2. Create Users
  const users = [];
  const passwordHash = await bcrypt.hash('password123', 10); // Default user password
  const adminPasswordHash = await bcrypt.hash('password', 10); // Admin password

  // Specific Admin User
  users.push({
    email: 'useradmin@example.com',
    username: 'useradmin',
    password: adminPasswordHash,
    name: 'user admin',
    bio: 'Platform Administrator',
    jobTitle: 'Admin',
    avatarUrl: faker.image.avatar(),
    isVerified: true,
    createdAt: new Date()
  });

  for (let i = 0; i < 20; i++) {
    const firstName = faker.person.firstName();
    const lastName = faker.person.lastName();
    const username = faker.internet.userName({ firstName, lastName }).toLowerCase().replace(/[^a-z0-9_]/g, '') + i;
    
    users.push({
      email: faker.internet.email({ firstName, lastName }),
      username: username,
      password: passwordHash,
      name: `${firstName} ${lastName}`,
      bio: faker.person.bio(),
      jobTitle: faker.person.jobTitle(),
      avatarUrl: faker.image.avatar(),
      isVerified: faker.datatype.boolean(0.2), // 20% verified
      createdAt: faker.date.past()
    });
  }

  const createdUsers = await prisma.user.createManyAndReturn({
    data: users
  });

  console.log(`✅ Created ${createdUsers.length} Users`);

  // 3. Create Social Links
  const socialPlatforms = ['INSTAGRAM', 'FACEBOOK', 'GITHUB', 'X', 'LINKEDIN', 'DRIBBBLE'];
  const socialLinks = [];

  for (const user of createdUsers) {
    // 50% chance to have social links
    if (Math.random() > 0.5) {
      const numLinks = faker.number.int({ min: 1, max: 3 });
      const platforms = faker.helpers.arrayElements(socialPlatforms, numLinks);

      for (const platform of platforms) {
        socialLinks.push({
          userId: user.id,
          social: platform,
          link: faker.internet.url(),
          username: user.username
        });
      }
    }
  }

  await prisma.socialLink.createMany({ data: socialLinks });
  console.log(`🔗 Created ${socialLinks.length} Social Links`);

  // 4. Skip creating follows and notifications per request
  console.log('⏭️ Skipping creation of follow relationships and notifications as requested');

  console.log('✨ Seeding Completed!');
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
