import { ILogRecord } from "@opentelemetry/otlp-transformer";

export interface ReportLog {
	time: Date;
	severity?: string;
	body: string;
	record: ILogRecord;
}
