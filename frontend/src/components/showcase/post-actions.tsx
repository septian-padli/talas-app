// components/home/molecules/post-actions.tsx

"use client";

import { cn } from "@/lib/utils";
import {
  HeartIcon,
  ChatBubbleLeftIcon,
  BookmarkIcon,
  ShareIcon,
} from "@heroicons/react/24/outline";
import {
  HeartIcon as HeartIconSolid,
  BookmarkIcon as BookmarkIconSolid,
} from "@heroicons/react/24/solid";
import ActionButton from "./ActionButton";

interface PostActionsProps {
  likes: number;
  comments: number;
  onLikeToggle: () => void;
  onComment: (e: React.MouseEvent) => void; // Changed from () => void to (e: React.MouseEvent) => void
  onShare: () => void;
  isLiked: boolean;
  isBookmarked: boolean;
  onBookmarkToggle: () => void;
  variant?: 'full' | 'bookmark-only';
}

export function PostActions({
  likes,
  comments,
  onLikeToggle,
  onComment,
  onShare,
  isLiked,
  isBookmarked,
  onBookmarkToggle,
  variant = 'full',
}: PostActionsProps) {

  if (variant === 'bookmark-only') {
    return (
      // The parent div can be simpler as the button controls its width and content alignment.
      // 'flex' ensures it behaves as a flex item if PostActions itself is in a flex layout.
      <div className="flex items-center w-full"> {/* Ensure this div also takes full width if needed or rely on PostCard's layout */}
        <ActionButton
          icon={isBookmarked ? (
            <BookmarkIconSolid className="w-5 h-5" />
          ) : (
            <BookmarkIcon className="w-5 h-5" />
          )}
          label={isBookmarked ? "Unsave" : "Save"}
          onClick={onBookmarkToggle}
          active={isBookmarked}
          fullWidth={true} // Pass fullWidth prop to the ActionButton
        />
      </div>
    );
  }

  // Full actions for other pages (remains the same)
  return (
    <div className="flex justify-between items-center">
      <ActionButton
        icon={isLiked ? (
          <HeartIconSolid className="w-5 h-5" />
        ) : (
          <HeartIcon className="w-5 h-5" />
        )}
        label={`${likes} Likes`}
        onClick={onLikeToggle}
        active={isLiked}
      />
      <ActionButton
        icon={<ChatBubbleLeftIcon className="w-5 h-5" />}
        label={`${comments} Comments`}
        onClick={onComment}
      />
      <ActionButton
        icon={isBookmarked ? (
          <BookmarkIconSolid className="w-5 h-5" />
        ) : (
          <BookmarkIcon className="w-5 h-5" />
        )}
        label="Save"
        onClick={onBookmarkToggle}
        active={isBookmarked}
      />
      <ActionButton
        icon={<ShareIcon className="w-5 h-5" />}
        label="Share"
        onClick={onShare}
      />
    </div>
  );
}