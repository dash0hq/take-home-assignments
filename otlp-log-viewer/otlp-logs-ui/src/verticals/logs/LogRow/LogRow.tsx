import React from "react";
import { ReportLog } from "@/types/logs";
import styles from "./LogRow.module.css";

interface Props {
	log: ReportLog;
}

const LogRow: React.FC<Props> = ({ log }) => {
	return (
		<div className={styles.logRow} role="row">
			<span role="cell">{log.severity}</span>
			<span role="cell">{log.time.toUTCString()}</span>
			<span role="cell">{log.body}</span>
		</div>
	);
};

export default React.memo(LogRow);
