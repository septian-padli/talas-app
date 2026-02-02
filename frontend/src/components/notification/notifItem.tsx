import React from "react";
import {
  HeartIcon,
  ChatBubbleOvalLeftIcon,
  UserPlusIcon,
  BellIcon,
} from "@heroicons/react/24/solid";
import { formatDistanceToNow } from "date-fns";
import { NotificationItem, NotificationType } from "@/types/notification";
import Image from "next/image";

// const getNotificationIcon = (notification: NotificationType) => {
//   const { type, title } = notification;
//   const iconProps = { className: "w-5 h-5 text-green-500" };

//   switch (type?.toLowerCase()) {
//     case 'like':
//     case 'like_project':
//       return <HeartIcon {...iconProps} />;
//     case 'comment':
//     case 'comment_project':
//       return <ChatBubbleOvalLeftIcon {...iconProps} />;
//     case 'follow':
//     case 'follow_user':
//       return <UserPlusIcon {...iconProps} />;
//     default: {
//       const titleLower = title?.toLowerCase() || '';
//       if (titleLower.includes('liked') || titleLower.includes('like')) {
//         return <HeartIcon {...iconProps} />;
//       } else if (titleLower.includes('comment') || titleLower.includes('replied')) {
//         return <ChatBubbleOvalLeftIcon {...iconProps} />;
//       } else if (titleLower.includes('follow') || titleLower.includes('started following')) {
//         return <UserPlusIcon {...iconProps} />;
//       }
//       return <BellIcon {...iconProps} />;
//     }
//   }
// };


// contoh text type like: Budi Santoso menyukai showcase "Redesign Aplikasi Gojek"
// contoh text type comment: Siti Aminah mengomentari "Website E-Commerce": "Keren banget bang, tech stack-nya..."
// contoh text type follow: Reza Rahadian mulai mengikuti Anda
// contoh text type collaborator_accepted: Dian Sastro sekarang kolaborator di showcase "Sistem Manajemen Gudang"

// function getTextContent
function getTextContent(notification: NotificationItem): string {
  const actor = notification.data?.actor?.username || "Seseorang";
  const entityTitle = notification.data?.entity?.title || "";
  const commentText = notification.data?.preview?.text || "";
  switch (notification.type) {
    case NotificationType.LIKE:
      if (entityTitle) {
        return `${actor} menyukai showcase "${entityTitle}"`;
      }
      return `${actor} menyukai postingan Anda`;
    case NotificationType.COMMENT:
      if (entityTitle && commentText) {
        return `${actor} mengomentari "${entityTitle}": "${commentText.length > 60 ? commentText.slice(0, 60) + '...' : commentText}"`;
      } else if (entityTitle) {
        return `${actor} mengomentari "${entityTitle}"`;
      }
      return `${actor} mengomentari postingan Anda`;
    case NotificationType.FOLLOW:
      return `${actor} mulai mengikuti Anda`;
    case NotificationType.COLLABORATOR_ACCEPTED:
      if (entityTitle) {
        return `${actor} sekarang kolaborator di showcase "${entityTitle}"`;
      }
      return `${actor} sekarang menjadi kolaborator Anda`;
    default:
      // fallback untuk tipe string lain
      if (typeof notification.type === "string") {
        const typeLower = notification.type.toLowerCase();
        if (typeLower.includes("like")) {
          return `${actor} menyukai postingan Anda`;
        } else if (typeLower.includes("comment")) {
          return `${actor} mengomentari postingan Anda`;
        } else if (typeLower.includes("follow")) {
          return `<span classname="font-semibold">${actor}</span> mulai mengikuti Anda`;
        } else if (typeLower.includes("collab")) {
          return `${actor} sekarang menjadi kolaborator Anda`;
        }
      }
      return "Ada notifikasi baru";
  }
}


export function NotificationItems({ notification }: { notification: NotificationItem }) {
  return (
    <div
      className={`px-4 py-3 flex items-start transition-all border-b border-gray-700 last:border-b-0 ${notification.is_read ? "opacity-70" : ""}`}
    >
      <div className="shrink-0 mt-1 mr-4">
        {notification.data?.actor?.avatar_url ? (
          <Image
            width={64}
            height={64}
            src={notification.data.actor.avatar_url}
            alt="avatar"
            className="w-8 h-8 md:w-10 md:h-10 rounded-full object-cover border border-gray-500"
          />
        ) : (
          <BellIcon className="w-5 h-5 md:w-10 md:h-10 text-gray-400" />
        )}
      </div>
      <div className="flex-1">
        <p className={`text-normal ${!notification.is_read ? "font-bold" : "font-medium"}`}>
          {getTextContent(notification)}
        </p>
        {/* button follow back */}

      </div>
      <div className="text-xs text-gray-400 ml-2">
        {formatDistanceToNow(new Date(notification.created_at), { addSuffix: false })}
      </div>
    </div>
  );
}
