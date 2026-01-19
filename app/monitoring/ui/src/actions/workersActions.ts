import { Dispatch } from "redux";
import { listWorkers, ListWorkersResponse } from "../api";
import { toErrorString, toErrorStringWithHttpStatus } from "../utils";

// List of worker related action types.
export const LIST_WORKERS_BEGIN = "LIST_WORKERS_BEGIN";
export const LIST_WORKERS_SUCCESS = "LIST_WORKERS_SUCCESS";
export const LIST_WORKERS_ERROR = "LIST_WORKERS_ERROR";

interface ListWorkersBeginAction {
  type: typeof LIST_WORKERS_BEGIN;
}
interface ListWorkersSuccessAction {
  type: typeof LIST_WORKERS_SUCCESS;
  payload: ListWorkersResponse;
}
interface ListWorkersErrorAction {
  type: typeof LIST_WORKERS_ERROR;
  error: string; // error description
}

// Union of all worker related actions.
export type WorkersActionTypes =
  | ListWorkersBeginAction
  | ListWorkersSuccessAction
  | ListWorkersErrorAction;

export function listWorkersAsync() {
  return async (dispatch: Dispatch<WorkersActionTypes>) => {
    dispatch({ type: LIST_WORKERS_BEGIN });
    try {
      const response = await listWorkers();
      dispatch({
        type: LIST_WORKERS_SUCCESS,
        payload: response,
      });
    } catch (error) {
      console.error(`listWorkersAsync: ${toErrorStringWithHttpStatus(error)}`);
      dispatch({
        type: LIST_WORKERS_ERROR,
        error: toErrorString(error),
      });
    }
  };
}
