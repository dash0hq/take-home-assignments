"use client";
import React, { useCallback, useState } from "react";
import { ReportLog } from "@/types/logs";
import classNames from "classnames";
import styles from "./LogRow.module.css";

interface Props {
	log: ReportLog;
}

const LogRow: React.FC<Props> = ({ log }) => {
	const [isExpanded, setIsExpanded] = useState(false);

	const handleClick = useCallback(() => {
		setIsExpanded((v) => !v);
	}, []);

	return (
		<>
			<div
				className={classNames(
					styles.logRow,
					isExpanded && styles.active
				)}
				role="row"
				onClick={handleClick}
			>
				<span role="cell">{log.severity}</span>
				<span role="cell">{log.time.toUTCString()}</span>
				<span role="cell">{log.body}</span>
			</div>
			{isExpanded && <pre>{JSON.stringify(log.record, null, 2)}</pre>}
		</>
	);
};

export default React.memo(LogRow);
