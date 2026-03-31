import { startTransition, useEffect, useRef, useState } from "react";

type Movie = {
  id: string;
  title: string;
  rows: number;
  seats_per_row: number;
};

type SeatStatus = {
  seat_id: string;
  booked: boolean;
  confirmed: boolean;
  user_id?: string;
};

type HoldResponse = {
  session_id: string;
  movie_id: string;
  seat_id: string;
  expires_at: string;
};

type ActiveSession = {
  sessionID: string;
  movieID: string;
  seatID: string;
  expiresAt: number;
};

type FlashMessage = {
  kind: "success" | "error" | "info";
  text: string;
};

const API_BASE = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");
const POLL_INTERVAL_MS = 2000;

const legend = [
  { label: "Available", tone: "available" },
  { label: "Your hold", tone: "held-mine" },
  { label: "Other hold", tone: "held-other" },
  { label: "Confirmed", tone: "confirmed" },
] as const;

function App() {
  const [userID] = useState(() => crypto.randomUUID().replace(/-/g, "").slice(0, 12));
  const [movies, setMovies] = useState<Movie[]>([]);
  const [selectedMovie, setSelectedMovie] = useState<Movie | null>(null);
  const [seatStatuses, setSeatStatuses] = useState<SeatStatus[]>([]);
  const [activeSession, setActiveSession] = useState<ActiveSession | null>(null);
  const [flashMessage, setFlashMessage] = useState<FlashMessage | null>(null);
  const [moviesLoading, setMoviesLoading] = useState(true);
  const [seatsLoading, setSeatsLoading] = useState(false);
  const [pendingSeatID, setPendingSeatID] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const statusTimeoutRef = useRef<number | null>(null);

  async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers: {
        "Content-Type": "application/json",
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (response.status === 204) {
      return null as T;
    }

    const data: unknown = await response.json();

    if (!response.ok) {
      const errorMessage =
        typeof data === "object" &&
        data !== null &&
        "error" in data &&
        typeof data.error === "string"
          ? data.error
          : "Request failed";

      throw new Error(errorMessage);
    }

    return data as T;
  }

  const showFlashMessage = useEvent((message: FlashMessage | null, persist = false) => {
    if (statusTimeoutRef.current) {
      window.clearTimeout(statusTimeoutRef.current);
      statusTimeoutRef.current = null;
    }

    setFlashMessage(message);

    if (message && !persist) {
      statusTimeoutRef.current = window.setTimeout(() => {
        setFlashMessage(null);
        statusTimeoutRef.current = null;
      }, 3200);
    }
  });

  const loadMovies = useEvent(async () => {
    setMoviesLoading(true);

    try {
      const data = await api<Movie[]>("GET", "/movies");
      startTransition(() => {
        setMovies(data);
        setSelectedMovie((currentMovie) => {
          if (!currentMovie) {
            return data[0] ?? null;
          }

          return data.find((movie) => movie.id === currentMovie.id) ?? data[0] ?? null;
        });
      });
    } catch (error) {
      showFlashMessage(
        {
          kind: "error",
          text: error instanceof Error ? error.message : "Could not load movies.",
        },
        true,
      );
    } finally {
      setMoviesLoading(false);
    }
  });

  const loadSeats = useEvent(async (movie: Movie | null) => {
    if (!movie) {
      setSeatStatuses([]);
      return;
    }

    setSeatsLoading(true);

    try {
      const data = await api<SeatStatus[]>("GET", `/movies/${movie.id}/seats`);
      startTransition(() => {
        setSeatStatuses(data);
      });
    } catch (error) {
      showFlashMessage(
        {
          kind: "error",
          text: error instanceof Error ? error.message : "Could not refresh seats.",
        },
        true,
      );
    } finally {
      setSeatsLoading(false);
    }
  });

  const releaseSession = useEvent(async (session: ActiveSession | null, quiet = false) => {
    if (!session) {
      return;
    }

    try {
      await api<null>("DELETE", `/sessions/${session.sessionID}`, { user_id: userID });
    } catch {
      if (!quiet) {
        showFlashMessage({ kind: "error", text: "Could not release the active hold." }, true);
      }
    }
  });

  useEffect(() => {
    void loadMovies();
  }, [loadMovies]);

  useEffect(() => {
    void loadSeats(selectedMovie);

    if (!selectedMovie) {
      return;
    }

    const interval = window.setInterval(() => {
      void loadSeats(selectedMovie);
    }, POLL_INTERVAL_MS);

    return () => {
      window.clearInterval(interval);
    };
  }, [selectedMovie, loadSeats]);

  useEffect(() => {
    if (!activeSession) {
      setNow(Date.now());
      return;
    }

    setNow(Date.now());

    const interval = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => {
      window.clearInterval(interval);
    };
  }, [activeSession]);

  useEffect(() => {
    return () => {
      if (statusTimeoutRef.current) {
        window.clearTimeout(statusTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!activeSession) {
      return;
    }

    if (activeSession.expiresAt > now) {
      return;
    }

    setActiveSession(null);
    showFlashMessage({ kind: "error", text: "Your hold expired." });

    if (selectedMovie) {
      void loadSeats(selectedMovie);
    }
  }, [activeSession, now, selectedMovie, loadSeats, showFlashMessage]);

  async function handleMovieSelect(movie: Movie) {
    if (movie.id === selectedMovie?.id) {
      return;
    }

    const previousSession = activeSession;
    setActiveSession(null);
    setPendingSeatID(null);

    if (previousSession) {
      await releaseSession(previousSession, true);
    }

    startTransition(() => {
      setSelectedMovie(movie);
      setSeatStatuses([]);
      setFlashMessage(null);
    });
  }

  async function handleSeatHold(seatID: string) {
    if (!selectedMovie || activeSession || pendingSeatID) {
      return;
    }

    setPendingSeatID(seatID);

    try {
      const data = await api<HoldResponse>("POST", `/movies/${selectedMovie.id}/seats/${seatID}/hold`, {
        user_id: userID,
      });

      setActiveSession({
        sessionID: data.session_id,
        movieID: data.movie_id,
        seatID: data.seat_id,
        expiresAt: new Date(data.expires_at).getTime(),
      });

      showFlashMessage({ kind: "info", text: `Seat ${seatID} is now on hold for you.` });
      await loadSeats(selectedMovie);
    } catch (error) {
      showFlashMessage({
        kind: "error",
        text: error instanceof Error ? error.message : "Could not hold that seat.",
      });
    } finally {
      setPendingSeatID(null);
    }
  }

  async function handleConfirm() {
    if (!activeSession || !selectedMovie) {
      return;
    }

    try {
      await api<null>("PUT", `/sessions/${activeSession.sessionID}/confirm`, {
        user_id: userID,
      });

      setActiveSession(null);
      showFlashMessage({ kind: "success", text: `Seat ${activeSession.seatID} confirmed.` }, true);
      await loadSeats(selectedMovie);
    } catch (error) {
      showFlashMessage({
        kind: "error",
        text: error instanceof Error ? error.message : "Could not confirm the seat.",
      });
    }
  }

  async function handleRelease() {
    if (!activeSession || !selectedMovie) {
      return;
    }

    const session = activeSession;
    setActiveSession(null);

    try {
      await api<null>("DELETE", `/sessions/${session.sessionID}`, {
        user_id: userID,
      });

      showFlashMessage({ kind: "info", text: `Seat ${session.seatID} released.` });
      await loadSeats(selectedMovie);
    } catch (error) {
      setActiveSession(session);
      showFlashMessage({
        kind: "error",
        text: error instanceof Error ? error.message : "Could not release the seat.",
      });
    }
  }

  const remainingSeconds = activeSession
    ? Math.max(0, Math.floor((activeSession.expiresAt - now) / 1000))
    : 0;
  const remainingMinutesLabel = String(Math.floor(remainingSeconds / 60)).padStart(2, "0");
  const remainingSecondsLabel = String(remainingSeconds % 60).padStart(2, "0");

  const seatMap = new Map(seatStatuses.map((seat) => [seat.seat_id, seat]));
  const availableSeatCount = selectedMovie
    ? selectedMovie.rows * selectedMovie.seats_per_row - seatStatuses.filter((seat) => seat.booked || seat.confirmed).length
    : 0;
  const occupiedSeatCount = selectedMovie
    ? selectedMovie.rows * selectedMovie.seats_per_row - availableSeatCount
    : 0;

  return (
    <div className="page-shell">
      <div className="page-glow page-glow--left" />
      <div className="page-glow page-glow--right" />

      <main className="app-shell">
        <section className="hero">
          <div>
            <p className="eyebrow">Realtime reservations</p>
            <h1>Cinema Booking</h1>
            <p className="hero-copy">
              Pick a show, watch the room update live, and lock a seat before the timer runs out.
            </p>
          </div>

          <div className="hero-meta">
            <div className="meta-pill">
              <span className="meta-label">User</span>
              <strong>{userID}</strong>
            </div>
            <div className="meta-pill">
              <span className="meta-label">Active hold</span>
              <strong>{activeSession?.seatID ?? "None"}</strong>
            </div>
          </div>
        </section>

        <section className="movie-strip">
          <div className="section-heading">
            <div>
              <p className="section-kicker">Showtimes</p>
              <h2>Choose a movie</h2>
            </div>
            {moviesLoading ? <span className="section-note">Loading titles...</span> : null}
          </div>

          <div className="movie-grid">
            {movies.map((movie) => {
              const isSelected = selectedMovie?.id === movie.id;
              const capacity = movie.rows * movie.seats_per_row;

              return (
                <button
                  key={movie.id}
                  type="button"
                  className={`movie-card ${isSelected ? "movie-card--selected" : ""}`}
                  onClick={() => void handleMovieSelect(movie)}
                >
                  <span className="movie-card__badge">{String(capacity).padStart(2, "0")} seats</span>
                  <h3>{movie.title}</h3>
                  <p>
                    {movie.rows} rows x {movie.seats_per_row} seats per row
                  </p>
                </button>
              );
            })}
          </div>
        </section>

        {selectedMovie ? (
          <section className="dashboard">
            <div className="screen-panel panel">
              <div className="section-heading">
                <div>
                  <p className="section-kicker">Auditorium</p>
                  <h2>{selectedMovie.title}</h2>
                </div>
                <div className="stats">
                  <div className="stat-chip">
                    <span>Available</span>
                    <strong>{availableSeatCount}</strong>
                  </div>
                  <div className="stat-chip">
                    <span>Occupied</span>
                    <strong>{occupiedSeatCount}</strong>
                  </div>
                </div>
              </div>

              <div className="screen-frame">
                <div className="screen-label">Screen</div>
                <div className="screen-bar" />

                <div className={`seat-grid ${seatsLoading ? "seat-grid--loading" : ""}`}>
                  {buildSeatRows(selectedMovie).map((row) => (
                    <div
                      className="seat-row"
                      key={row.rowLabel}
                      style={{
                        gridTemplateColumns: `1.6rem repeat(${selectedMovie.seats_per_row}, minmax(2.2rem, 1fr)) 1.6rem`,
                      }}
                    >
                      <span className="row-label">{row.rowLabel}</span>

                      {row.seats.map((seatID, index) => {
                        const info = seatMap.get(seatID);
                        const isConfirmed = info?.confirmed ?? false;
                        const isHeldByUser = Boolean(info?.booked && info.user_id === userID && !info.confirmed);
                        const isHeldByOther = Boolean(info?.booked && info.user_id !== userID && !info.confirmed);
                        const isActionable = !isConfirmed && !isHeldByOther && !isHeldByUser && !activeSession;
                        const isPending = pendingSeatID === seatID;

                        return (
                          <button
                            key={seatID}
                            type="button"
                            className={[
                              "seat",
                              isHeldByUser ? "seat--held-mine" : "",
                              isHeldByOther ? "seat--held-other" : "",
                              isConfirmed ? "seat--confirmed" : "",
                              isPending ? "seat--pending" : "",
                            ]
                              .filter(Boolean)
                              .join(" ")}
                            onClick={() => void handleSeatHold(seatID)}
                            disabled={!isActionable || isPending}
                            aria-label={`Seat ${seatID}`}
                          >
                            <span>{index + 1}</span>
                          </button>
                        );
                      })}

                      <span className="row-label">{row.rowLabel}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="legend">
                {legend.map((item) => (
                  <div key={item.tone} className="legend-item">
                    <span className={`legend-swatch legend-swatch--${item.tone}`} />
                    {item.label}
                  </div>
                ))}
              </div>
            </div>

            <aside className="checkout-panel panel">
              <div className="section-heading">
                <div>
                  <p className="section-kicker">Checkout</p>
                  <h2>{activeSession ? "Complete your hold" : "Waiting for a seat"}</h2>
                </div>
              </div>

              {activeSession ? (
                <>
                  <div className="checkout-summary">
                    <div>
                      <span>Seat</span>
                      <strong>{activeSession.seatID}</strong>
                    </div>
                    <div>
                      <span>Movie</span>
                      <strong>{selectedMovie.title}</strong>
                    </div>
                    <div>
                      <span>Session</span>
                      <strong>{activeSession.sessionID.slice(0, 8)}...</strong>
                    </div>
                  </div>

                  <div className={`timer-card ${remainingSeconds < 60 ? "timer-card--urgent" : ""}`}>
                    <span>Time remaining</span>
                    <strong>
                      {remainingMinutesLabel}:{remainingSecondsLabel}
                    </strong>
                  </div>

                  <div className="action-row">
                    <button type="button" className="action-btn action-btn--confirm" onClick={() => void handleConfirm()}>
                      Confirm seat
                    </button>
                    <button type="button" className="action-btn action-btn--release" onClick={() => void handleRelease()}>
                      Release
                    </button>
                  </div>
                </>
              ) : (
                <div className="empty-card">
                  <p>Select any available seat to create a temporary hold and start checkout.</p>
                </div>
              )}

              {flashMessage ? (
                <div className={`status-banner status-banner--${flashMessage.kind}`}>{flashMessage.text}</div>
              ) : null}
            </aside>
          </section>
        ) : (
          <section className="panel empty-state">
            {moviesLoading ? (
              <p>Loading the cinema map...</p>
            ) : (
              <p>No movies are available yet. Once the API returns shows, they will appear here.</p>
            )}
          </section>
        )}
      </main>
    </div>
  );
}

function buildSeatRows(movie: Movie) {
  const rowLabels = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

  return Array.from({ length: movie.rows }, (_, rowIndex) => {
    const rowLabel = rowLabels[rowIndex] ?? `R${rowIndex + 1}`;

    return {
      rowLabel,
      seats: Array.from({ length: movie.seats_per_row }, (_, seatIndex) => `${rowLabel}${seatIndex + 1}`),
    };
  });
}

export default App;

function useEvent<T extends (...args: never[]) => unknown>(callback: T): T {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const stableCallbackRef = useRef<T | null>(null);

  if (stableCallbackRef.current === null) {
    stableCallbackRef.current = ((...args: Parameters<T>) => callbackRef.current(...args)) as T;
  }

  return stableCallbackRef.current;
}
