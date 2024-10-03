import type { Metadata } from 'next';
import { PropsWithChildren } from 'react';


export const metadata: Metadata = {
    title: 'OTLP Logs UI',
    description: 'API server exposing OTLP resourceLogs as JSON',
}

export default function RootLayout({children}: Readonly<PropsWithChildren>) {
    return (
            <html lang="en">
            <body>{children}</body>
            </html>
    )
}
