import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import Link from "next/link";
import { SiDribbble, SiFacebook, SiGithub, SiInstagram, SiX } from '@icons-pack/react-simple-icons';
import { Linkedin } from "lucide-react";
import { useProfile } from "@/hooks/useProfile";

const getIcon = (platform: string) => {
  switch (platform) {
    case 'INSTAGRAM':
      return <SiInstagram className="w-6 h-6" />;
    case 'FACEBOOK':
      return <SiFacebook className="w-6 h-6" />;
    case 'GITHUB':
      return <SiGithub className="w-6 h-6" />;
    case 'X':
      return <SiX className="w-6 h-6" />;
    case 'LINKEDIN':
      return <Linkedin className="w-6 h-6" />;
    case 'DRIBBBLE':
      return <SiDribbble className="w-6 h-6" />;
    default:
      return null;
  }
}

const ProfileCard: React.FC = () => {
  // data profile dari hook useProfile nanti
  const { data: profileResponse, isLoading, isError } = useProfile();
  const profile = profileResponse?.data.user;

  if (isLoading) {
    return <div>Loading profile...</div>;
  }

  if (isError || !profile) {
    return <div>Error loading profile.</div>;
  }

  return (
    <>
      {/* header profile */}
      <div className="flex justify-between md:gap-8">
        <div>
          <h2 className="text-white text-2xl font-bold leading-[1.3] mb-1">{profile?.name}</h2>
          <h2 className="text-white/75 text-base leading-[1.3] mb-4">@{profile?.username}</h2>
          <p className="text-white text-base font-bold leading-normal mb-4">{profile?.job_title}</p>
          <p className="text-white text-sm font-normal leading-tight">
            {profile?.bio}
          </p>
        </div>
        {/* avatar */}
        <div className="flex items-center justify-center">
          <Avatar className="w-16 md:w-20 lg:w-22 xl:w-24 h-16 md:h-20 lg:h-22 xl:h-24">
            <AvatarImage src={profile?.avatar_url || undefined} alt={`@${profile?.username}`} />
            <AvatarFallback className="text-2xl font-bold">{profile?.name ? profile.name.charAt(0) + profile.name.charAt(1) : "CN"}</AvatarFallback>
          </Avatar>
        </div>
      </div>

      {/* social */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mt-6 md:mt-8">
        {/* follow/following */}
        <div className="flex items-center gap-2 text-white/80 text-sm">
          <span>{profile?.followers_count} Pengikut</span>
          <span className="mx-2 hidden md:inline-block">|</span>
          <span>{profile?.following_count} Diikuti</span>
        </div>
        {/* social icons */}
        <div className="flex gap-4 items-center justify-start md:justify-end">
          {(Array.isArray(profile?.social_links) ? profile.social_links : []).map((link, index) => (
            <Link
              key={index}
              href={link.link || "#"}
              className="text-white/80 hover:text-white transition-colors"
              target="_blank"
              rel="noopener noreferrer"
            >
              {getIcon(link.social)}
            </Link>
          ))}
        </div>
      </div>

      {/* count showcase */}
      <div className="w-full border-y border-white/10 py-4 mt-6 md:mt-8 flex items-center justify-center">
        <p>10 Showcase</p>
      </div>
    </>
  )
}

export default ProfileCard