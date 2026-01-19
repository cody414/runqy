import React from "react";
import { connect, ConnectedProps } from "react-redux";
import Container from "@material-ui/core/Container";
import { makeStyles } from "@material-ui/core/styles";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import Typography from "@material-ui/core/Typography";
import Alert from "@material-ui/lab/Alert";
import AlertTitle from "@material-ui/lab/AlertTitle";
import WorkersTable from "../components/WorkersTable";
import { listWorkersAsync } from "../actions/workersActions";
import { AppState } from "../store";
import { usePolling } from "../hooks";

const useStyles = makeStyles((theme) => ({
  container: {
    paddingTop: theme.spacing(4),
    paddingBottom: theme.spacing(4),
  },
  paper: {
    padding: theme.spacing(2),
    display: "flex",
    overflow: "auto",
    flexDirection: "column",
  },
  heading: {
    paddingLeft: theme.spacing(2),
    marginBottom: theme.spacing(1),
  },
  description: {
    paddingLeft: theme.spacing(2),
    marginBottom: theme.spacing(2),
    color: theme.palette.text.secondary,
  },
}));

function mapStateToProps(state: AppState) {
  return {
    loading: state.workers.loading,
    error: state.workers.error,
    workers: state.workers.data,
    pollInterval: state.settings.pollInterval,
  };
}

const connector = connect(mapStateToProps, { listWorkersAsync });

type Props = ConnectedProps<typeof connector>;

function WorkersView(props: Props) {
  const { pollInterval, listWorkersAsync } = props;
  const classes = useStyles();

  usePolling(listWorkersAsync, pollInterval);

  return (
    <Container maxWidth="lg" className={classes.container}>
      <Grid container spacing={3}>
        {props.error === "" ? (
          <Grid item xs={12}>
            <Paper className={classes.paper} variant="outlined">
              <Typography variant="h6" className={classes.heading}>
                Workers
              </Typography>
              <Typography variant="body2" className={classes.description}>
                External workers (e.g., RunPod) that process tasks from queues.
                Workers send heartbeats every 5 seconds. Stale workers haven't sent a heartbeat in 30+ seconds.
              </Typography>
              <WorkersTable workers={props.workers} />
            </Paper>
          </Grid>
        ) : (
          <Grid item xs={12}>
            <Alert severity="error">
              <AlertTitle>Error</AlertTitle>
              Could not retrieve workers data —{" "}
              <strong>See the logs for details</strong>
            </Alert>
          </Grid>
        )}
      </Grid>
    </Container>
  );
}

export default connector(WorkersView);
