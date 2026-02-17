"use client";

import { PostCard } from "@/components/showcase/post-card";

import ProfileCard from "@/components/profile/profile-card";
import { useProfile } from "@/hooks/useProfile";
import { useHeaderStore } from "@/store/useHeaderStore";
import { ShowcaseDetail } from "@/types/showcase";
import { useEffect } from "react";



interface ProfilePageProps {
    prop: string;
}

const dummyPosts: ShowcaseDetail[] = [
    {
        "id": "66f310ae-3805-49ef-bf0a-fd7dac0e1df0",
        "created_at": "2026-02-17T18:29:44.79499+07:00",
        "updated_at": "2026-02-17T18:29:44.79499+07:00",
        "deleted_at": null,
        "title": "project roro jongrang",
        "slug": "project-roro-jongrang-8f849af6",
        "content": "project roro jongrang",
        "tags": [
            "nextjs",
            " golang"
        ],
        "is_edited": false,
        "views_count": 4,
        "likes_count": 0,
        "comments_count": 0,
        "shares_count": 0,
        "category_id": "8e11bcf5-68a2-400b-a99c-2158f330cc62",
        "category": {
            "id": "8e11bcf5-68a2-400b-a99c-2158f330cc62",
            "created_at": "2026-02-09T10:33:43.779686+07:00",
            "updated_at": "2026-02-09T10:33:43.779686+07:00",
            "name": "Backend Engineering",
            "slug": "backend-engineering"
        },
        "media": [
            {
                "id": "a73a3d23-685d-4a18-a081-f41142043c8a",
                "created_at": "2026-02-17T18:29:44.809852+07:00",
                "updated_at": "2026-02-17T18:29:44.809852+07:00",
                "deleted_at": null,
                "showcase_id": "66f310ae-3805-49ef-bf0a-fd7dac0e1df0",
                "url": "https://res.cloudinary.com/dqtea12yq/image/upload/v1771327777/talas/showcases/Wallpaper%20UPI.png.png",
                "type": "IMAGE",
                "position": 1
            },
            {
                "id": "2f6c933d-d190-4e24-ba74-45ca1a883057",
                "created_at": "2026-02-17T18:29:44.809852+07:00",
                "updated_at": "2026-02-17T18:29:44.809852+07:00",
                "deleted_at": null,
                "showcase_id": "66f310ae-3805-49ef-bf0a-fd7dac0e1df0",
                "url": "https://res.cloudinary.com/dqtea12yq/image/upload/v1771327782/talas/showcases/night.jpg.jpg",
                "type": "IMAGE",
                "position": 2
            },
            {
                "id": "612a210f-8bcc-4b6f-b15b-c2f195f63729",
                "created_at": "2026-02-17T18:29:44.809852+07:00",
                "updated_at": "2026-02-17T18:29:44.809852+07:00",
                "deleted_at": null,
                "showcase_id": "66f310ae-3805-49ef-bf0a-fd7dac0e1df0",
                "url": "https://res.cloudinary.com/dqtea12yq/image/upload/v1771327784/talas/showcases/Wallpaper%20logo.png.png",
                "type": "IMAGE",
                "position": 3
            }
        ],
        "collaborators": [
            {
                "id": "8281b5e5-6d43-4a88-9705-4daece90ffd5",
                "role": "OWNER",
                "status": "ACCEPTED",
                "user": {
                    "id": "dbd34860-bea1-4421-a3d0-33dd0428fbf1",
                    "name": "user admin",
                    "username": "useradmin",
                    "jobTitle": "Admin",
                    "avatarUrl": "https://res.cloudinary.com/dqtea12yq/image/upload/v1770617782/talas/avatars/avatar_1770617781403.jpg"
                }
            }
        ]
    }
];

const ProfilePage: React.FC<ProfilePageProps> = () => {
    const { setTitle } = useHeaderStore();
    useEffect(() => {
        setTitle("Profile");
        return () => setTitle("Talas App");
    }, [setTitle]);

    const { data: profileResponse, isLoading, isError } = useProfile();
    const profile = profileResponse?.data.user;

    if (isLoading) {
        return <div className="p-8 text-center">Loading profile...</div>;
    }
    if (isError || !profile) {
        return <div className="p-8 text-center text-red-500">Error loading profile.</div>;
    }

    return (
        <div className="bg-[#181818] rounded-2xl">
            <div className="px-6 py-8">
                <ProfileCard profile={profile} />

                {/* Showcases */}
                {dummyPosts.map((post) => (
                    <div key={post.id} className="border-b border-white/10 p-1">
                        <PostCard
                            showcase={post}
                        />
                    </div>
                ))}
            </div>
        </div>
    );
};

export default ProfilePage;
