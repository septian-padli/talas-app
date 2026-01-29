const prisma = require('../utils/prisma');

// 1. Get Bulk Users (untuk Feed / Content Service)
const getBulkUsers = async (req, res) => {
  try {
    const { userIds } = req.body;

    if (!userIds || !Array.isArray(userIds) || userIds.length === 0) {
      return res.status(400).json({
        code: 400,
        success: false,
        message: 'Invalid input: userIds must be a non-empty array'
      });
    }

    const users = await prisma.user.findMany({
      where: {
        id: { in: userIds }
      },
      select: {
        id: true,
        username: true,
        name: true,
        avatarUrl: true,
        jobTitle: true,
        isVerified: true
      }
    });

    res.json({
      code: 200,
      success: true,
      data: users
    });

  } catch (error) {
    console.error('Internal BulkUsers Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Internal Server Error'
    });
  }
};

// 1.5 Get Bulk Users By Username (untuk Invite Collaborator)
const getBulkUsersByUsername = async (req, res) => {
  try {
    const { usernames } = req.body;

    if (!usernames || !Array.isArray(usernames) || usernames.length === 0) {
      return res.status(400).json({
        code: 400,
        success: false,
        message: 'Invalid input: usernames must be a non-empty array'
      });
    }

    const users = await prisma.user.findMany({
      where: {
        username: { in: usernames }
      },
      select: {
        id: true,
        username: true,
        name: true,
        avatarUrl: true,
        jobTitle: true,
        isVerified: true
      }
    });

    res.json({
      code: 200,
      success: true,
      data: users
    });

  } catch (error) {
    console.error('Internal BulkUsersByUsername Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Internal Server Error'
    });
  }
};

// 2. Get User Follower IDs (untuk Filter Feed)
const getUserFollowers = async (req, res) => {
  try {
    const { id } = req.params;
    const limit = parseInt(req.query.limit) || 1000;
    const cursor = req.query.cursor;

    // Decode Cursor
    let cursorObj = undefined;
    if (cursor) {
      try {
        cursorObj = JSON.parse(Buffer.from(cursor, 'base64').toString('ascii'));
      } catch (e) {
        // Invalid cursor
      }
    }

    const followers = await prisma.follow.findMany({
      where: { followingId: id },
      select: { 
        id: true, // Needed for cursor
        followerId: true 
      },
      take: limit + 1,
      cursor: cursorObj ? { id: cursorObj.id } : undefined,
      skip: cursorObj ? 1 : 0,
      orderBy: [
        { createdAt: 'desc' },
        { id: 'desc' }
      ]
    });

    let hasNext = false;
    let nextCursor = null;

    if (followers.length > limit) {
      hasNext = true;
      followers.pop();
      const lastItem = followers[followers.length - 1];
      
      const nextCursorData = { id: lastItem.id };
      nextCursor = Buffer.from(JSON.stringify(nextCursorData)).toString('base64');
    }

    // Map object array [{followerId: "..."}] -> ["...", "..."]
    const followerIds = followers.map(f => f.followerId);

    res.json({
      code: 200,
      success: true,
      data: {
        follower_ids: followerIds,
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
    console.error('Internal GetUserFollowers Error:', error);
    res.status(500).json({
      code: 500,
      success: false,
      message: 'Internal Server Error',
      errors: null
    });
  }
};

module.exports = {
  getBulkUsers,
  getBulkUsersByUsername,
  getUserFollowers
};
