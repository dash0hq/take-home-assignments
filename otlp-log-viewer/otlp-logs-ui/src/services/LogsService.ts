import { IExportLogsServiceRequest } from "@opentelemetry/otlp-transformer";
import { ReportLog } from "@/types/logs";
import { LOGS_API } from "@/config";
import otlpLogsToReportLogs from "@/adapters/otlpLogsToReportLogs";

let logRecords: ReportLog[] | undefined = undefined;

const LogsService = {
	async getLogs(): Promise<IExportLogsServiceRequest> {
		const response = await fetch(`${LOGS_API}/logs`, {
			cache: "force-cache",
		});
		return response.json();
	},

	async getReportLogs() {
		if (!logRecords) {
			logRecords = otlpLogsToReportLogs(await this.getLogs());
		}

		return logRecords;
	},
};

export default LogsService;
