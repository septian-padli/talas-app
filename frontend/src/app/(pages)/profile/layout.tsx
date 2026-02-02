
export default function ProfileLayout({ children }: { children: React.ReactNode }) {
    return (
        <div className="max-w-2/3 mx-auto">
            <div className="mb-8">
                <h1 className="text-white text-center font-bold text-3xl leading-[225%] font-comfortaa">
                    Profile
                </h1>
            </div>

            {children}
        </div>
    );
}