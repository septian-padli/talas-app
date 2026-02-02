// src/providers/QueryProvider.tsx
'use client'; // Wajib karena menggunakan Context API

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

export default function QueryProvider({ children }: { children: React.ReactNode }) {
    // Pastikan QueryClient dibuat sekali saja per session browser
    const [queryClient] = useState(
        () =>
            new QueryClient({
                defaultOptions: {
                    queries: {
                        // Data dianggap "fresh" selama 1 menit (tidak refetch otomatis)
                        staleTime: 60 * 1000,
                        // Retry 1 kali saja jika gagal (agar tidak spam error)
                        retry: 1,
                        refetchOnWindowFocus: false, // Opsional: Matikan auto-refetch saat pindah tab
                    },
                },
            })
    );

    return (
        <QueryClientProvider client={queryClient}>
            {children}
        </QueryClientProvider>
    );
}