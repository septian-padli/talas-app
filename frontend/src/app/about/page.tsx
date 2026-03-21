"use client";

import { Button } from "@/components/ui/button";
import Image from "next/image";


const AboutPage = () => {
    // Daftar logo techstack
    const techstackLogos = [
        { src: "/logo/techstack/figma.svg", alt: "Figma" },
        { src: "/logo/techstack/java.svg", alt: "Java" },
        { src: "/logo/techstack/js.svg", alt: "JavaScript" },
        { src: "/logo/techstack/php.svg", alt: "PHP" },
        { src: "/logo/techstack/python.svg", alt: "Python" },
    ];
    return (
        <>
            {/* hero section */}
            <section className="py-8 md:py-12 min-h-screen flex items-center flex-col justify-center gap-12 relative">
                <h1 className="text-6xl w-3/4 leading-none text-center font-semibold">SHOWCASE AND COLLABORATE ON YOUR NEXT BIG PROJECT</h1>
                <p className="text-white">Talas is a platform to showcase your projects, receive feedback, and collaborate with other creators.</p>
                <Button variant={"brand"} size={"lg"}>Get Started</Button>

                <div className="absolute bottom-8 w-full flex justify-center gap-6">
                    {techstackLogos.map((logo) => (
                        <Image key={logo.src} src={logo.src} alt={logo.alt} width={200} height={60} className="h-16" />
                    ))}
                    {techstackLogos.map((logo) => (
                        <Image key={logo.src} src={logo.src} alt={logo.alt} width={200} height={60} className="h-16" />
                    ))}
                </div>
            </section>

            <section className="py-8 md:py-12 min-h-screen relative container flex flex-col justify-center items-center">
                <h2 className="text-3xl text-center text-white font-semibold mb-8">A Glimpse into the Talas Experience</h2>

                <div className="mx-auto w-3/4 relative">
                    <span className="block absolute -top-0.5 -left-0.5 w-full h-full rounded-3xl -z-10 bg-brand-700-75 blur-xl opacity-70" />
                    <Image src={"/img/homepage.png"} alt="Talas Homepage" width={800} height={600} className="rounded-3xl w-full translate-0.5" />
                </div>
            </section>

            <section className="py-8 md:py-12 min-h-screen relative container flex flex-col justify-center items-center">
                <h2 className="text-3xl text-center text-white font-semibold mb-8">Behind Talas</h2>
                <p>Meet the minds behind Talas—a team of developers and designers passionate about collaboration, innovation, and building a thriving tech community</p>

                <div className="flex flex-row gap-6">
                    <div className="overflow-hidden aspect-square rounded-xl shadow shadow-brand-700 bg-amber-100 w-xs relative">
                        <Image src={"/img/dummy/profile-photo-dummy.jpg"} alt="Profile" width={400} height={400} className="w-full h-full object-cover" />

                        {/* gradient to top, #000000 to transparent */}
                        <div className="bg-">
                            <p>Muhammad Padli Septiana</p>
                            <p>Fullstack Developer</p>
                        </div>
                    </div>
                </div>
            </section>
        </>
    );
};

export default AboutPage;
