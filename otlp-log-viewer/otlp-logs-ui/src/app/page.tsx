import LogsTable from "@/verticals/logs/LogsTable";
import LogsHistogram from "@/verticals/logs/LogsHistogram";
import "./global.css";

export default function Home() {
	return (
		<main>
			<h1>OTLP Logs UI</h1>
			<LogsHistogram />
			<LogsTable />
		</main>
	);
}
