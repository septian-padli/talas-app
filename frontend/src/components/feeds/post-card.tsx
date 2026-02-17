// components/home/organisms/post-card.tsx

"use client";

import Image from "next/image";
import Link from "next/link";
import { useState, useEffect } from "react";
import { Github, Figma } from "lucide-react";
import { useRouter } from "next/navigation";
import { formatDistanceToNow } from 'date-fns';
import { PostHeader } from "./post-header";
import { PostActions } from "./post-actions";
import { toast } from "sonner";
import { ShowcaseDetail } from "@/types/showcase";


// Accepts a showcase object directly

interface PostCardProps {
  showcase: ShowcaseDetail;
  displayContext?: 'saved-page' | string;
}

export function PostCard({ showcase, displayContext }: PostCardProps) {
  // 1. Only show owner collaborator (safe for undefined)
  const owner = showcase.collaborators?.find((c) => c.role === 'OWNER');
  const username = owner?.user.username || '';
  const userRole = owner?.role || '';
  const avatarSrc = owner?.user.avatar_url || '';

  // 2. Only IMAGE media, sorted by position
  const images = (showcase.media || [])
    .filter((m) => m.type == 'IMAGE')
    .sort((a, b) => a.position - b.position)
    .map((m) => m.url);

  // 3. Default handlers
  const onToggleLike = () => console.log('Like toggled', showcase.id);
  const onToggleBookmark = () => console.log('Bookmark toggled', showcase.id);

  // 4. Category mapping
  const category = showcase.category
    ? { slug: showcase.category.slug, title: showcase.category.name }
    : undefined;

  // 5. Timestamp
  const timestamp = showcase.created_at;

  // 6. Link figma/github kosong
  const link_figma = undefined;
  const link_github = undefined;

  // 7. Title & content
  const title = showcase.title;
  const content = showcase.content;

  // 8. Slug
  const slug = showcase.slug;

  // 9. Likes/comments
  const likes = showcase.likes_count;
  const comments = showcase.comments_count;

  // 10. Dummy state for like/bookmark (for now always false)
  const isLiked = false;
  const isBookmarked = false;
  const router = useRouter();
  const [isMobile, setIsMobile] = useState(false);

  const allDisplayImages = images;

  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth <= 690);
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  // const handleCardClick = (e: React.MouseEvent) => {
  //   const target = e.target as HTMLElement;
  //   // Prevents navigation if a link or button within the card is clicked
  //   if (target.closest('a, button, [data-prevent-card-click="true"]')) {
  //     return;
  //   }
  //   if (slug) {
  //     router.push(`/project/${slug}`);
  //   } else {
  //     router.push(`/project/${id}`);
  //   }
  // };

  const handleCommentClick = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    if (slug) {
      router.push(`/project/${slug}#comments`);
    } else {
      router.push(`/project/${showcase.id}#comments`);
    }
  };

  const displayTitle = title || (content ? content.split('\n')[0] : 'Untitled Project');
  const displayContent = title ? content : (content ? content.split('\n').slice(1).join('\n') : '');

  let formattedTimestamp = 'just now';
  if (timestamp) {
    try {
      formattedTimestamp = formatDistanceToNow(new Date(timestamp), { addSuffix: true });
    } catch (error) {
      console.error("Failed to format timestamp:", timestamp, error);
      formattedTimestamp = new Date(timestamp).toLocaleDateString();
    }
  }

  const postActionsVariant = displayContext === 'saved-page' ? 'bookmark-only' : 'full';

  const handleShare = () => {
    const projectUrl = `${window.location.origin}/project/${slug || showcase.id}`;
    navigator.clipboard.writeText(projectUrl)
      .then(() => {
        toast.success("Project link copied to clipboard!", { position: "bottom-right" });
      })
      .catch(err => {
        console.error('Failed to copy link: ', err);
        toast.error("Failed to copy project link", { position: "bottom-right" });
      });
  };

  return (
    <div
      className={`p-4 ${isMobile ? 'bg-background' : ''}`}
    >
      <PostHeader
        username={username}
        userRole={userRole}
        avatarSrc={avatarSrc}
        timestamp={formattedTimestamp}
      />

      <div className="mb-6">
        {/* This div acts as a block-level container for the title link */}
        <div>
          <Link
            href={`/project/${slug || showcase.id}`}
            onClick={(e) => e.stopPropagation()}
            data-prevent-card-click="true"
          >
            <h2 className="text-lg font-bold hover:text-primary transition-colors duration-200 inline-block">
              {title || (content ? content.split('\n')[0] : 'Untitled Project')}
            </h2>
          </Link>
        </div>
        {category && (
          // Ensure the category is also a block-level element
          <p className="text-sm text-muted-foreground mb-3">
            {category.title}
          </p>
        )}
        <p className="text-white text-sm whitespace-pre-line">
          {title ? content : (content ? content.split('\n').slice(1).join('\n') : '')}
        </p>

        {/* Figma/Github links hidden for now */}
      </div>

      {allDisplayImages.length > 0 && (
        <div
          className={`grid gap-2 mb-4 ${allDisplayImages.length === 1 ? 'grid-cols-1' :
            allDisplayImages.length === 2 ? 'grid-cols-2' :
              allDisplayImages.length === 3 ? 'grid-cols-3' :
                allDisplayImages.length === 4 ? 'grid-cols-2' : 'grid-cols-3'
            }`}
          onClick={e => e.stopPropagation()} // Prevent card click
          data-prevent-card-click="true"
        >
          {allDisplayImages.slice(0, 5).map((imagePath, index) => {
            return (
              <div
                key={index}
                className={`aspect-video bg-muted rounded-md overflow-hidden relative ${(allDisplayImages.length === 3 && index === 0) ? 'md:col-span-3 row-span-2' :
                  (allDisplayImages.length === 5 && index === 0) ? 'col-span-3 md:col-span-2 row-span-2' :
                    (allDisplayImages.length === 5 && (index === 1 || index === 2)) ? 'col-span-1 md:col-span-1 row-span-1' :
                      (allDisplayImages.length > 3 && index > 0 && allDisplayImages.length % 2 !== 0 && index === allDisplayImages.length - 1) ? 'col-span-2' :
                        ''
                  } ${isMobile && allDisplayImages.length > 2 && index === 0 ? 'col-span-full' : ''}`}
              >
                <Image
                  src={imagePath}
                  alt={`${title || 'Project'} image ${index + 1}`}
                  fill
                  className="object-cover"
                  sizes="(max-width: 690px) 100vw, (max-width: 1024px) 50vw, 33vw"
                  onError={(e) => {
                    const target = e.target as HTMLImageElement;
                    target.src = '/img/dummy/project-photo-dummy.jpg';
                  }}
                />
              </div>
            );
          })}
        </div>
      )}

      <div className="pt-2" onClick={e => e.stopPropagation()} data-prevent-card-click="true">
        <PostActions
          likes={likes}
          comments={comments}
          onLikeToggle={onToggleLike}
          onComment={handleCommentClick}
          onShare={handleShare}
          isLiked={isLiked}
          isBookmarked={isBookmarked}
          onBookmarkToggle={onToggleBookmark}
          variant={postActionsVariant}
        />
      </div>
    </div>
  );
}