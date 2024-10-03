import LogsTable from "@/verticals/logs/LogsTable";
import "./global.css";

export default function Home() {
	return (
		<main>
			<h1>OTLP Logs UI</h1>
			<LogsTable />
		</main>
	);
}
