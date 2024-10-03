import type { Metadata } from "next";
import { PropsWithChildren } from "react";

export const metadata: Metadata = {
	title: "OTLP Logs UI",
};

export default function RootLayout({ children }: Readonly<PropsWithChildren>) {
	return (
		<html lang="en">
			<body>{children}</body>
		</html>
	);
}
