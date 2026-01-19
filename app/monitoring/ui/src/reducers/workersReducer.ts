import {
  LIST_WORKERS_BEGIN,
  LIST_WORKERS_ERROR,
  LIST_WORKERS_SUCCESS,
  WorkersActionTypes,
} from "../actions/workersActions";
import { WorkerInfo } from "../api";

interface WorkersState {
  loading: boolean;
  error: string;
  data: WorkerInfo[];
}

const initialState: WorkersState = {
  loading: false,
  error: "",
  data: [],
};

export default function workersReducer(
  state = initialState,
  action: WorkersActionTypes
): WorkersState {
  switch (action.type) {
    case LIST_WORKERS_BEGIN:
      return {
        ...state,
        loading: true,
      };

    case LIST_WORKERS_SUCCESS:
      return {
        loading: false,
        error: "",
        data: action.payload.workers,
      };

    case LIST_WORKERS_ERROR:
      return {
        ...state,
        error: action.error,
        loading: false,
      };

    default:
      return state;
  }
}
