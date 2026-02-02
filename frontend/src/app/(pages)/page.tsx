"use client";
import { PostCard } from "@/components/feeds/post-card";
import { PostComposer } from "@/components/feeds/post-composer";

interface HomePageProps {
    prop: string;
}

const dummyPosts = [
    {
        id: "1",
        slug: "my-first-project",
        title: "My First Project",
        username: "johndoe",
        userRole: "Developer",
        avatarSrc: "https://randomuser.me/api/portraits/men/1.jpg",
        timestamp: new Date().toISOString(),
        content: "This is my first project!\nIt uses Next.js and Tailwind.",
        images: [
            "https://images.unsplash.com/photo-1465101046530-73398c7f28ca?auto=format&fit=crop&w=600&q=80",
            "https://images.unsplash.com/photo-1506744038136-46273834b3fb?auto=format&fit=crop&w=600&q=80",
            "https://images.unsplash.com/photo-1519125323398-675f0ddb6308?auto=format&fit=crop&w=600&q=80"
        ],
        likes: 12,
        comments: 3,
        link_figma: "https://figma.com/file/abc123",
        link_github: "https://github.com/johndoe/project1",
        isLiked: false,
        isBookmarked: false,
        onToggleLike: () => { },
        onToggleBookmark: () => { },
        category: { slug: "web", title: "Web App" },
    },
    {
        id: "2",
        title: "UI Exploration",
        username: "janedoe",
        userRole: "Designer",
        avatarSrc: "https://randomuser.me/api/portraits/women/2.jpg",
        timestamp: new Date(Date.now() - 3600 * 1000).toISOString(),
        content: "UI/UX exploration for a mobile app.",
        images: [
            "https://images.unsplash.com/photo-1519125323398-675f0ddb6308?auto=format&fit=crop&w=600&q=80"
        ],
        likes: 8,
        comments: 1,
        isLiked: true,
        isBookmarked: true,
        onToggleLike: () => { },
        onToggleBookmark: () => { },
        category: { slug: "design", title: "Design" },
    },
    {
        id: "3",
        title: "Open Source Contribution",
        username: "alice",
        userRole: "Contributor",
        avatarSrc: "https://randomuser.me/api/portraits/women/3.jpg",
        timestamp: new Date(Date.now() - 86400 * 1000).toISOString(),
        content: "Contributed to an open source library.",
        images: [
            "https://images.unsplash.com/photo-1519125323398-675f0ddb6308?auto=format&fit=crop&w=600&q=80"
        ],
        likes: 20,
        comments: 5,
        isLiked: false,
        isBookmarked: false,
        onToggleLike: () => { },
        onToggleBookmark: () => { },
        category: { slug: "oss", title: "Open Source" },
    },
];

const HomePage: React.FC<HomePageProps> = () => {
    return (
        <>
            <div className="max-w-2/3 mx-auto">

                {/* composer */}
                <div className="mb-8">
                    <PostComposer />
                </div>

                <div className="bg-[#181818] rounded-2xl">
                    {dummyPosts.map((post) => (
                        <div key={post.id} className="border-b border-white/10 p-1">
                            <PostCard
                                {...post}
                            />
                        </div>
                    ))}
                </div>
            </div>

        </>
    );
};

export default HomePage;
