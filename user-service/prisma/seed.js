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

  // 4. Create Follows (Relationships)
  const follows = [];
  
  for (const user of createdUsers) {
    const potentialFollowings = createdUsers.filter(u => u.id !== user.id);
    const followingList = faker.helpers.arrayElements(potentialFollowings, faker.number.int({ min: 0, max: 8 }));

    for (const followedUser of followingList) {
      follows.push({
        followerId: user.id,
        followingId: followedUser.id,
        createdAt: faker.date.recent()
      });

      // Update counters manually (since we use createMany, Prisma middleware might not trigger if any)
      // But for bulk seed, usually we accept slight divergence or update manually. 
      // Here we will rely on a potential separate fix or just let counters be re-calculated if needed.
      // Ideally we should update counters, but for simplicity in seeding large data, we might skip precise counters 
      // or update them after. Let's try to be precise.
    }
  }

  // Bulk Insert Follows
  await prisma.follow.createMany({ data: follows });
  
  // Recalculate and Update Counters
  console.log('🔄 Updating Follow Counters...');
  for (const user of createdUsers) {
    const followers = follows.filter(f => f.followingId === user.id).length;
    const following = follows.filter(f => f.followerId === user.id).length;

    await prisma.user.update({
      where: { id: user.id },
      data: {
        followersCount: followers,
        followingCount: following
      }
    });
  }
  
  console.log(`👥 Created ${follows.length} Follow relationships`);

  // 5. Create Notifications (Mock Data)
  const notifications = [];
  const notifTypes = ['LIKE', 'COMMENT', 'FOLLOW', 'COLLABORATOR_INVITED'];

  for (const user of createdUsers) {
    // 30% chance user has notifications
    if (Math.random() > 0.3) {
      const numNotifs = faker.number.int({ min: 1, max: 5 });
      for (let i = 0; i < numNotifs; i++) {
        const sender = faker.helpers.arrayElement(createdUsers.filter(u => u.id !== user.id));
        const type = faker.helpers.arrayElement(notifTypes);
        
        notifications.push({
          userId: user.id,
          senderId: sender.id,
          type: type,
          title: `New ${type} from ${sender.name}`,
          content: faker.lorem.sentence(),
          isRead: faker.datatype.boolean(),
          createdAt: faker.date.recent()
        });
      }
    }
  }

  await prisma.notification.createMany({ data: notifications });
  console.log(`🔔 Created ${notifications.length} Notifications`);

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
