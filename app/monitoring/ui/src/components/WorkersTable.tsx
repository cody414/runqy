import React from "react";
import { makeStyles } from "@material-ui/core/styles";
import Table from "@material-ui/core/Table";
import TableBody from "@material-ui/core/TableBody";
import TableCell from "@material-ui/core/TableCell";
import TableContainer from "@material-ui/core/TableContainer";
import TableHead from "@material-ui/core/TableHead";
import TableRow from "@material-ui/core/TableRow";
import Chip from "@material-ui/core/Chip";
import { WorkerInfo } from "../api";
import { timeAgo } from "../utils";

const useStyles = makeStyles((theme) => ({
  table: {
    minWidth: 650,
  },
  staleChip: {
    backgroundColor: theme.palette.error.main,
    color: theme.palette.error.contrastText,
  },
  runningChip: {
    backgroundColor: theme.palette.success.main,
    color: theme.palette.success.contrastText,
  },
  stoppedChip: {
    backgroundColor: theme.palette.grey[500],
    color: theme.palette.common.white,
  },
}));

interface Props {
  workers: WorkerInfo[];
}

export default function WorkersTable(props: Props) {
  const classes = useStyles();
  const { workers } = props;

  if (workers.length === 0) {
    return (
      <TableContainer>
        <Table className={classes.table}>
          <TableHead>
            <TableRow>
              <TableCell>Worker ID</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Queues</TableCell>
              <TableCell>Concurrency</TableCell>
              <TableCell>Started</TableCell>
              <TableCell>Last Heartbeat</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <TableRow>
              <TableCell colSpan={6} align="center">
                No workers found. Workers will appear here when they connect and send heartbeats.
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    );
  }

  const getStatusChip = (worker: WorkerInfo) => {
    if (worker.is_stale) {
      return <Chip label="Stale" size="small" className={classes.staleChip} />;
    }
    if (worker.status === "running") {
      return <Chip label="Running" size="small" className={classes.runningChip} />;
    }
    if (worker.status === "stopped") {
      return <Chip label="Stopped" size="small" className={classes.stoppedChip} />;
    }
    return <Chip label={worker.status || "Unknown"} size="small" />;
  };

  return (
    <TableContainer>
      <Table className={classes.table}>
        <TableHead>
          <TableRow>
            <TableCell>Worker ID</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Queues</TableCell>
            <TableCell>Concurrency</TableCell>
            <TableCell>Started</TableCell>
            <TableCell>Last Heartbeat</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {workers.map((worker) => (
            <TableRow key={worker.worker_id}>
              <TableCell>{worker.worker_id}</TableCell>
              <TableCell>{getStatusChip(worker)}</TableCell>
              <TableCell>{worker.queues}</TableCell>
              <TableCell>{worker.concurrency}</TableCell>
              <TableCell>
                {worker.started_at > 0
                  ? timeAgo(new Date(worker.started_at * 1000).toISOString())
                  : "N/A"}
              </TableCell>
              <TableCell>
                {worker.last_beat > 0
                  ? timeAgo(new Date(worker.last_beat * 1000).toISOString())
                  : "N/A"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
