const prisma = require('../utils/prisma');
const { publishEvent } = require('../utils/rabbitmq');
const { updateProfileSchema } = require('../validations/userValidation');


const getMyProfile = async (req, res) => {
  try {
    // req.user has been attached by the protect middleware
    // We fetch fresh data from DB to ensure we have the latest counts and details
    if (!req.user || !req.user.id) {
        return res.status(401).json({
            code: 401,
            success: false,
            message: 'Unauthorized: User tidak ditemukan',
            errors: null
        });
    }

    const user = await prisma.user.findUnique({
      where: { id: req.user.id },
      select: {
        id: true,
        username: true,
        email: true,
        name: true,
        bio: true,
        avatarUrl: true,
        jobTitle: true,
        followersCount: true,
        followingCount: true,
        createdAt: true,
        updatedAt: true
      }
    });

    if (!user) {
      return res.status(404).json({
        code: 404,
        success: false,
        message: 'User tidak ditemukan',
        errors: null
      });
    }

    // Map to API Contract (snake_case)
    const responseData = {
      user: {
        id: user.id,
        username: user.username,
        email: user.email,
        name: user.name,
        bio: user.bio,
        avatar_url: user.avatarUrl,
        is_verified: false, // Default value as per plan
        followers_count: user.followersCount,
        following_count: user.followingCount,
        created_at: user.createdAt,
        updated_at: user.updatedAt
      }
    };

    res.json({
      code: 200,
      success: true,
      data: responseData,
      errors: null
    });

  } catch (error) {
    console.error('GetMyProfile Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

const getUserByUsername = async (req, res) => {
  try {
    const { username } = req.params;
    const currentUserId = req.user.id;

    const user = await prisma.user.findUnique({
      where: { username },
      select: {
        id: true,
        username: true,
        name: true,
        bio: true,
        avatarUrl: true,
        followersCount: true,
        followingCount: true
      }
    });

    if (!user) {
      return res.status(404).json({
        code: 404,
        success: false,
        message: 'User tidak ditemukan',
        errors: null
      });
    }

    // Check is_following status
    const isFollowing = await prisma.follow.findUnique({
      where: {
        followerId_followingId: {
          followerId: currentUserId,
          followingId: user.id
        }
      }
    });

    const responseData = {
      user: {
        id: user.id,
        username: user.username,
        name: user.name,
        bio: user.bio,
        avatar_url: user.avatarUrl,
        followers_count: user.followersCount,
        following_count: user.followingCount,
        is_following: !!isFollowing
      }
    };

    res.json({
      code: 200,
      success: true,
      data: responseData,
      errors: null
    });

  } catch (error) {
    console.error('GetUserByUsername Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

const toggleFollow = async (req, res) => {
  const followerId = req.user.id;
  const followingId = req.params.id;

  // 1. Validasi Dasar
  if (followerId === followingId) {
    return res.status(400).json({
      code: 400,
      success: false,
      message: 'Anda tidak dapat mengikuti diri sendiri',
      errors: null
    });
  }

  // DEBUG: Check IDs
  // console.log(`[ToggleFollow] Debugging...`);
  // console.log(`[ToggleFollow] Follower ID (Me): ${followerId}`);
  // console.log(`[ToggleFollow] Following ID (Target): ${followingId}`);

  // Check if both users actually exist before transaction to isolate the error
  const checkFollower = await prisma.user.findUnique({ where: { id: followerId } });
  const checkFollowing = await prisma.user.findUnique({ where: { id: followingId } });

  if (!checkFollower) {
    console.error(`[ToggleFollow] Error: Follower (Me) with ID ${followerId} does NOT exist in DB.`);
    return res.status(404).json({
      code: 404,
      success: false,
      message: 'User pengikut (Anda) tidak ditemukan di database. Token mungkin invalid/user terhapus.',
      errors: null
    });
  }

  if (!checkFollowing) {
    console.error(`[ToggleFollow] Error: Following (Target) with ID ${followingId} does NOT exist in DB.`);
    return res.status(404).json({
      code: 404,
      success: false,
      message: 'User target tidak ditemukan',
      errors: null
    });
  }

  try {
    const result = await prisma.$transaction(async (tx) => {
      // Langkah 1 (READ): Cek status follow saat ini
      const existingFollow = await tx.follow.findUnique({
        where: {
          followerId_followingId: {
            followerId,
            followingId
          }
        }
      });

      let status = '';
      let isFollowing = false;

      // Langkah 2 (WRITE): Execute updates
      if (existingFollow) {
        // UNFOLLOW
        status = 'UNFOLLOWED';
        isFollowing = false;

        await Promise.all([
          tx.follow.delete({
            where: { id: existingFollow.id }
          }),
          tx.user.update({
            where: { id: followerId },
            data: { followingCount: { decrement: 1 } }
          }),
          tx.user.update({
            where: { id: followingId },
            data: { followersCount: { decrement: 1 } }
          })
        ]);
      } else {
        // FOLLOW
        status = 'FOLLOWED';
        isFollowing = true;

        await Promise.all([
          tx.follow.create({
            data: {
              followerId,
              followingId
            }
          }),
          tx.user.update({
            where: { id: followerId },
            data: { followingCount: { increment: 1 } }
          }),
          tx.user.update({
            where: { id: followingId },
            data: { followersCount: { increment: 1 } }
          })
        ]);
      }

      return { status, isFollowing };
    });

    // RabbitMQ Publish (Fire and Forget)
    const eventData = {
      event: result.status,
      timestamp: new Date().toISOString(),
      actor: {
        id: req.user.id,
        username: req.user.username,
        name: req.user.name,
        avatar: req.user.avatarUrl
      },
      target: { id: followingId }
    };
    
    // Publish ke routing key: user.followed atau user.unfollowed
    publishEvent(`user.${result.status.toLowerCase()}`, eventData);

    res.json({
      code: 200,
      success: true,
      message: result.status === 'FOLLOWED' ? 'Berhasil mengikuti user' : 'Berhasil berhenti mengikuti user',
      data: {
        is_following: result.isFollowing
      },
      errors: null
    });

  } catch (error) {
    // Handling Foreign Key Constraint (User Target tidak ditemukan)
    if (error.code === 'P2003') {
      return res.status(404).json({
        code: 404,
        success: false,
        message: 'User target tidak ditemukan',
        errors: null
      });
    }

    console.error('ToggleFollow Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};



const getUserFollowers = async (req, res) => {
  const targetUserId = req.params.id;
  const loggedInUserId = req.user.id;
  
  // 1. Parsing Query Params
  const limit = parseInt(req.query.limit) || 10;
  const cursor = req.query.cursor;

  // 2. Decode Cursor
  let cursorObj = undefined;
  if (cursor) {
    try {
      cursorObj = JSON.parse(Buffer.from(cursor, 'base64').toString('ascii'));
    } catch (e) {
      // Invalid cursor, abaikan atau return bad request (kita abaikan saja start dari awal)
    }
  }

  try {
    // 3. Query Prisma (Follower List)
    const follows = await prisma.follow.findMany({
      where: {
        followingId: targetUserId
      },
      take: limit + 1, // Fetch +1 untuk cek next page
      cursor: cursorObj ? { id: cursorObj.id } : undefined,
      skip: cursorObj ? 1 : 0, // Skip cursor item itu sendiri
      orderBy: [
        { createdAt: 'desc' },
        { id: 'desc' }
      ],
      include: {
        follower: true // Ambil data user follower
      }
    });

    let hasNext = false;
    let nextCursor = null;

    if (follows.length > limit) {
      hasNext = true;
      follows.pop(); // Remove item ke-11
      const lastItem = follows[follows.length - 1];
      
      // Encode Next Cursor
      const nextCursorData = { id: lastItem.id };
      nextCursor = Buffer.from(JSON.stringify(nextCursorData)).toString('base64');
    }

    // 4. Batch Check "is_following" status (Optimized)
    // Cek apakah user yg login mem-follow para followers ini?
    const followerIds = follows.map(f => f.followerId);
    
    const myFollows = await prisma.follow.findMany({
      where: {
        followerId: loggedInUserId,
        followingId: {
          in: followerIds
        }
      },
      select: {
        followingId: true
      }
    });

    const myFollowsSet = new Set(myFollows.map(f => f.followingId));

    // 5. Mapping Response
    const mappedUsers = follows.map(item => ({
      id: item.follower.id,
      username: item.follower.username,
      name: item.follower.name,
      avatar_url: item.follower.avatarUrl,
      bio: item.follower.bio,
      is_following: myFollowsSet.has(item.follower.id),
      followed_at: item.createdAt.toISOString()
    }));

    res.json({
      code: 200,
      success: true,
      data: {
        users: mappedUsers,
        meta: {
          curr_cursor: cursor || null,
          next_cursor: nextCursor,
          has_next: hasNext,
          limit: limit
        }
      },
      errors: null
    });

  } catch (error) {
    console.error('GetFollowers Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

const getUserFollowing = async (req, res) => {
  const targetUserId = req.params.id;
  const loggedInUserId = req.user.id;
  
  // 1. Parsing Query Params
  const limit = parseInt(req.query.limit) || 10;
  const cursor = req.query.cursor;

  // 2. Decode Cursor
  let cursorObj = undefined;
  if (cursor) {
    try {
      cursorObj = JSON.parse(Buffer.from(cursor, 'base64').toString('ascii'));
    } catch (e) {
      // Invalid cursor
    }
  }

  try {
    // 3. Query Prisma (Following List)
    // "Siapa saja yang diikuti oleh targetUserId?"
    const follows = await prisma.follow.findMany({
      where: {
        followerId: targetUserId
      },
      take: limit + 1,
      cursor: cursorObj ? { id: cursorObj.id } : undefined,
      skip: cursorObj ? 1 : 0,
      orderBy: [
        { createdAt: 'desc' },
        { id: 'desc' }
      ],
      include: {
        following: true // Ambil data user yang diikuti
      }
    });

    let hasNext = false;
    let nextCursor = null;

    if (follows.length > limit) {
      hasNext = true;
      follows.pop();
      const lastItem = follows[follows.length - 1];
      
      const nextCursorData = { id: lastItem.id };
      nextCursor = Buffer.from(JSON.stringify(nextCursorData)).toString('base64');
    }

    // 4. Batch Check "is_following" status
    // List user yang ditampilkan (the users being followed by target)
    const displayedUserIds = follows.map(f => f.followingId);
    
    // Cek apakah user yg login mem-follow mereka?
    const myFollows = await prisma.follow.findMany({
      where: {
        followerId: loggedInUserId,
        followingId: {
          in: displayedUserIds
        }
      },
      select: {
        followingId: true
      }
    });

    const myFollowsSet = new Set(myFollows.map(f => f.followingId));

    // 5. Mapping Response
    const mappedUsers = follows.map(item => ({
      id: item.following.id,
      username: item.following.username,
      name: item.following.name,
      avatar_url: item.following.avatarUrl,
      bio: item.following.bio,
      is_following: myFollowsSet.has(item.following.id),
      followed_at: item.createdAt.toISOString()
    }));

    res.json({
      code: 200,
      success: true,
      data: {
        users: mappedUsers,
        meta: {
          curr_cursor: cursor || null,
          next_cursor: nextCursor,
          has_next: hasNext,
          limit: limit
        }
      },
      errors: null
    });

  } catch (error) {
    console.error('GetFollowing Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};


const updateUserProfile = async (req, res) => {
  try {
    const userId = req.user.id;

    // 1. Validasi Input
    const validation = updateProfileSchema.safeParse(req.body);
    if (!validation.success) {
      const errorDetails = validation.error.issues.map(err => ({
        field: err.path.join('.'), // Join path for nested objects (e.g. socialLinks.0.link)
        message: err.message
      }));

      return res.status(400).json({
        code: 400,
        success: false,
        message: 'Validasi gagal',
        errors: errorDetails
      });
    }

    const { name, bio, jobTitle, avatarUrl, socialLinks } = validation.data;

    // 2. Transaction Update
    // Kita gunakan transaction untuk update User info & Replace Social Links (Flush & Fill strategy)
    const result = await prisma.$transaction(async (tx) => {
      // A. Update User Info
      const updatedUser = await tx.user.update({
        where: { id: userId },
        data: {
          name,
          bio,
          jobTitle,
          avatarUrl
        }
      });

      // B. Update Social Links (jika dikirim)
      if (socialLinks) {
        // Hapus link lama
        await tx.socialLink.deleteMany({
          where: { userId }
        });

        // Insert link baru
        if (socialLinks.length > 0) {
          await tx.socialLink.createMany({
            data: socialLinks.map(link => ({
              userId,
              social: link.social,
              link: link.link,
              username: link.username
            }))
          });
        }
      }

      return updatedUser;
    });

    // 3. RabbitMQ Publish (Jika Identitas Visual Berubah)
    // Cek apakah identity penting berubah
    const isIdentityChanged = (name && name !== req.user.name) || 
                              (jobTitle && jobTitle !== req.user.jobTitle) || 
                              (avatarUrl && avatarUrl !== req.user.avatarUrl);

    if (isIdentityChanged) {
      const eventData = {
        event: 'PROFILE_UPDATED',
        timestamp: new Date().toISOString(),
        data: {
          id: result.id,
          username: result.username, // Immutable
          name: result.name,
          jobTitle: result.jobTitle,
          avatar: result.avatarUrl
        }
      };
      
      publishEvent('user.profile.updated', eventData);
    }

    // 4. Fetch Complete Data (with Social Links) untuk Response
    const finalUserData = await prisma.user.findUnique({
      where: { id: userId },
      include: {
        socialLinks: true
      }
    });

    // Map Response
    const responseData = {
      user: {
        id: finalUserData.id,
        username: finalUserData.username,
        email: finalUserData.email,
        name: finalUserData.name,
        bio: finalUserData.bio,
        job_title: finalUserData.jobTitle,
        avatar_url: finalUserData.avatarUrl,
        social_links: finalUserData.socialLinks.map(sl => ({
          social: sl.social,
          link: sl.link,
          username: sl.username
        }))
      }
    };

    res.json({
      code: 200,
      success: true,
      message: 'Profil berhasil diperbarui',
      data: responseData,
      errors: null
    });

  } catch (error) {
    console.error('UpdateProfile Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Terjadi kesalahan internal server',
      errors: null
    });
  }
};

module.exports = {
  getMyProfile,
  getUserByUsername,
  toggleFollow,
  getUserFollowers,
  getUserFollowing,
  updateUserProfile
};

