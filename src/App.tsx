import {
  Bell,
  Bot,
  ChevronRight,
  Clock3,
  Crown,
  Dice5,
  Flame,
  Gamepad2,
  LockKeyhole,
  MessageCircle,
  Play,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
  Trophy,
  UsersRound,
  Wifi
} from "lucide-react";
import type { CSSProperties } from "react";
import { useEffect, useMemo, useState } from "react";

type GameCategory = "블러핑" | "전략" | "순발력" | "숫자/조합" | "파티" | "카드";

interface GameCard {
  id: string;
  title: string;
  players: string;
  minPlayers: number;
  maxPlayers: number;
  time: string;
  difficulty: "쉬움" | "보통" | "어려움";
  categories: GameCategory[];
  accent: string;
}

const games: GameCard[] = [
  {
    id: "dalmuti",
    title: "위대한 달무티",
    players: "4-8명",
    minPlayers: 4,
    maxPlayers: 8,
    time: "20분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#c95f42"
  },
  {
    id: "werewolf",
    title: "한 밤의 늑대인간",
    players: "3-10명",
    minPlayers: 3,
    maxPlayers: 10,
    time: "10분",
    difficulty: "보통",
    categories: ["블러핑", "파티"],
    accent: "#6f5fbf"
  },
  {
    id: "rummikub",
    title: "루미큐브",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "35분",
    difficulty: "보통",
    categories: ["숫자/조합", "전략"],
    accent: "#2f78b7"
  },
  {
    id: "bang",
    title: "뱅!",
    players: "4-7명",
    minPlayers: 4,
    maxPlayers: 7,
    time: "40분",
    difficulty: "어려움",
    categories: ["블러핑", "카드"],
    accent: "#a3682a"
  },
  {
    id: "davinci",
    title: "다빈치 코드",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "15분",
    difficulty: "쉬움",
    categories: ["숫자/조합", "전략"],
    accent: "#1c8c8c"
  },
  {
    id: "halli-galli",
    title: "할리갈리",
    players: "2-6명",
    minPlayers: 2,
    maxPlayers: 6,
    time: "10분",
    difficulty: "쉬움",
    categories: ["순발력", "파티"],
    accent: "#d24f6a"
  },
  {
    id: "splendor",
    title: "스플랜더",
    players: "2-4명",
    minPlayers: 2,
    maxPlayers: 4,
    time: "30분",
    difficulty: "보통",
    categories: ["전략", "카드"],
    accent: "#2f8e74"
  }
];

const playerCount = 4;
const fallbackRecommendedGames = games.filter(
  (game) => game.minPlayers <= playerCount && game.maxPlayers >= playerCount
);

interface ApiGame {
  id: string;
  title: string;
  minPlayers: number;
  maxPlayers: number;
  estimatedMinutes: number;
  difficulty: "쉬움" | "보통" | "어려움";
  categories: GameCategory[];
}

interface GuestSession {
  sessionToken: string;
  user: {
    id: string;
    nickname: string;
  };
}

interface RoomParticipant {
  user: {
    id: string;
    nickname: string;
  };
  ready: boolean;
  host: boolean;
  seatIndex: number;
}

interface Room {
  id: string;
  code: string;
  gameId: string;
  status: "LOBBY" | "PLAYING" | "FINISHED" | "CLOSED";
  maxPlayers: number;
  participants: RoomParticipant[];
}

interface GameSession {
  id: string;
  roomId: string;
  gameId: string;
  status: "ACTIVE" | "FINISHED" | "ABORTED";
  state: {
    turnIndex?: number;
    round?: number;
    log?: string[];
    finished?: boolean;
  };
  results?: Array<{
    playerId: string;
    rank: number;
    score: number;
    outcome: "WIN" | "LOSE" | "DRAW";
  }>;
}

interface RealtimeMessage {
  room: string;
  type: string;
  payload: {
    room?: Room;
    session?: GameSession;
  };
}

const guestStorageKey = "board-table.guest-session";

export function App() {
  const [apiGames, setApiGames] = useState<ApiGame[]>([]);
  const [serverStatus, setServerStatus] = useState<"연결됨" | "오프라인 모드">("오프라인 모드");
  const [guestSession, setGuestSession] = useState<GuestSession | null>(() => readGuestSession());
  const [currentRoom, setCurrentRoom] = useState<Room | null>(null);
  const [currentSession, setCurrentSession] = useState<GameSession | null>(null);
  const [roomCodeInput, setRoomCodeInput] = useState("");
  const [roomMessage, setRoomMessage] = useState("방을 만들거나 초대 코드를 입력하세요.");

  useEffect(() => {
    const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";

    fetch(`${apiURL}/api/games/recommend?players=${playerCount}`)
      .then((response) => {
        if (!response.ok) throw new Error("recommendation request failed");
        return response.json() as Promise<{ games: ApiGame[] }>;
      })
      .then((data) => {
        setApiGames(data.games);
        setServerStatus("연결됨");
      })
      .catch(() => {
        setApiGames([]);
        setServerStatus("오프라인 모드");
      });
  }, []);

  useEffect(() => {
    const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
    const current = readGuestSession();

    if (current?.sessionToken) {
      fetch(`${apiURL}/api/me`, {
        headers: { Authorization: `Bearer ${current.sessionToken}` }
      })
        .then((response) => {
          if (!response.ok) throw new Error("guest session expired");
          return response.json() as Promise<{ user: GuestSession["user"] }>;
        })
        .then((data) => {
          const refreshed = { sessionToken: current.sessionToken, user: data.user };
          saveGuestSession(refreshed);
          setGuestSession(refreshed);
        })
        .catch(() => createGuest(apiURL).then(setGuestSession).catch(() => undefined));
      return;
    }

    createGuest(apiURL).then(setGuestSession).catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!currentRoom || !guestSession) return;

    const wsURL = import.meta.env.VITE_WS_URL ?? "ws://localhost:4000/ws";
    const socket = new WebSocket(
      `${wsURL}?room=${encodeURIComponent(`room:${currentRoom.id}`)}&user=${encodeURIComponent(
        guestSession.user.id
      )}`
    );

    socket.onopen = () => {
      setRoomMessage(`${currentRoom.code} 방에 실시간으로 연결되었습니다.`);
    };

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as RealtimeMessage;
      if (message.type !== "room.updated") return;

      const nextRoom = message.payload.room;
      if (!nextRoom) return;
      setCurrentRoom(nextRoom);
      setRoomMessage(`${nextRoom.code} 방 상태가 갱신되었습니다.`);
    };

    socket.onclose = () => {
      setRoomMessage("방 실시간 연결이 끊겼습니다. API 상태는 유지됩니다.");
    };

    return () => {
      socket.close();
    };
  }, [currentRoom?.id, guestSession]);

  useEffect(() => {
    if (!currentSession || !guestSession) return;

    const wsURL = import.meta.env.VITE_WS_URL ?? "ws://localhost:4000/ws";
    const socket = new WebSocket(
      `${wsURL}?room=${encodeURIComponent(`game:${currentSession.id}`)}&user=${encodeURIComponent(
        guestSession.user.id
      )}`
    );

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as RealtimeMessage;
      if (message.type !== "game.updated") return;
      if (!message.payload.session) return;

      setCurrentSession(message.payload.session);
    };

    return () => {
      socket.close();
    };
  }, [currentSession?.id, guestSession]);

  const recommendedGames = useMemo(() => {
    if (apiGames.length === 0) return fallbackRecommendedGames;

    return apiGames.map((apiGame) => {
      const fallback = games.find((game) => game.id === apiGame.id);
      return {
        id: apiGame.id,
        title: apiGame.title,
        players: `${apiGame.minPlayers}-${apiGame.maxPlayers}명`,
        minPlayers: apiGame.minPlayers,
        maxPlayers: apiGame.maxPlayers,
        time: `${apiGame.estimatedMinutes}분`,
        difficulty: apiGame.difficulty,
        categories: apiGame.categories,
        accent: fallback?.accent ?? "#2f8e74"
      };
    });
  }, [apiGames]);

  const activePlayerCount = currentRoom?.participants.length ?? playerCount;
  const partyMembers = currentRoom?.participants ?? [];
  const me = currentRoom?.participants.find(
    (participant) => participant.user.id === guestSession?.user.id
  );

  return (
    <main className="app-shell">
      <section className="hero-panel" aria-label="게임 로비">
        <nav className="top-bar">
          <div className="brand">
            <span className="brand-mark">
              <Dice5 size={22} aria-hidden="true" />
            </span>
            <div>
              <strong>Board Table</strong>
              <span>PWA 로비</span>
            </div>
          </div>
          <div className="top-actions">
            <button className="icon-button" aria-label="알림">
              <Bell size={19} />
            </button>
            <button className="profile-button" aria-label="내 프로필">
              {guestSession?.user.nickname.slice(-2) ?? "G"}
            </button>
          </div>
        </nav>

        <div className="hero-grid">
          <div className="hero-copy">
            <p className="eyebrow">
              <Wifi size={16} aria-hidden="true" />
              모바일, 태블릿, 데스크톱 동시 플레이 · 서버 {serverStatus}
            </p>
            <h1>친구와 바로 모이고, 인원에 맞는 게임을 즉시 시작하세요.</h1>
            <p className="summary">
              방 코드, 퀵매치, 재접속, 관전, 채팅까지 하나의 앱 화면에서 이어지는 보드게임 플랫폼입니다.
            </p>
            <div className="hero-actions">
              <button className="primary-button">
                <Play size={18} />
                퀵매치
              </button>
              <button className="secondary-button">
                <Plus size={18} />
                방 만들기
              </button>
            </div>
          </div>

          <div className="table-preview" aria-label="진행 중인 방 미리보기">
            <div className="table-felt">
              <span className="seat seat-a">민</span>
              <span className="seat seat-b">J</span>
              <span className="seat seat-c">S</span>
              <span className="seat seat-d">게</span>
              <div className="card-stack">
                <span />
                <span />
                <span />
              </div>
              <button className="bell-button" aria-label="할리갈리 종 치기">
                <Flame size={24} />
              </button>
            </div>
          </div>
        </div>
      </section>

      <section className="content-grid">
        <aside className="control-panel" aria-label="매칭 패널">
          <div className="room-card">
            <div className="section-title">
              <div>
                <span>{guestSession?.user.nickname ?? "게스트 준비 중"}</span>
              <h2>{activePlayerCount}명 입장 중</h2>
              </div>
              <UsersRound size={22} />
            </div>
            <div className="party-list">
              {(partyMembers.length > 0 ? partyMembers : demoParticipants()).map((participant) => (
                <div className="party-member" key={participant.user.id}>
                  <span>{participant.user.nickname.slice(0, 1)}</span>
                  <div>
                    <strong>{participant.user.nickname}</strong>
                    <small>{participant.host ? "방장" : participant.ready ? "준비 완료" : "대기 중"}</small>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="room-card">
            <div className="section-title">
              <div>
                <span>방 코드</span>
                <h2>{currentRoom?.code ?? "없음"}</h2>
              </div>
              <LockKeyhole size={22} />
            </div>
            <div className="room-actions">
              <button className="wide-button" onClick={() => createRoom(setCurrentRoom, setRoomMessage)}>
                방 만들기
                <ChevronRight size={18} />
              </button>
              <label className="code-input">
                <input
                  value={roomCodeInput}
                  onChange={(event) => setRoomCodeInput(event.target.value.toUpperCase())}
                  placeholder="초대 코드"
                  aria-label="초대 코드"
                />
              </label>
              <button
                className="wide-button dark"
                onClick={() => joinRoom(roomCodeInput, setCurrentRoom, setRoomMessage)}
              >
                코드 입장
                <ChevronRight size={18} />
              </button>
              {currentRoom ? (
                <button
                  className="wide-button ready"
                  onClick={() =>
                    setReady(currentRoom.id, !(me?.ready ?? false), setCurrentRoom, setRoomMessage)
                  }
                >
                  {me?.ready ? "준비 취소" : "Ready"}
                  <ChevronRight size={18} />
                </button>
              ) : null}
              {currentRoom ? (
                <button
                  className="wide-button play-now"
                  onClick={() => startGame(currentRoom.id, setCurrentRoom, setCurrentSession, setRoomMessage)}
                >
                  게임 시작
                  <ChevronRight size={18} />
                </button>
              ) : null}
            </div>
            <p className="room-message">{roomMessage}</p>
          </div>

          {currentSession ? (
            <div className="room-card">
              <div className="section-title">
                <div>
                  <span>게임 세션</span>
                  <h2>{currentSession.status === "FINISHED" ? "결과 확인" : "진행 중"}</h2>
                </div>
                <Gamepad2 size={22} />
              </div>
              <div className="game-session-panel">
                <p>라운드 {currentSession.state.round ?? 1}</p>
                <div className="session-log">
                  {(currentSession.state.log ?? []).slice(-3).map((item) => (
                    <span key={item}>{item}</span>
                  ))}
                </div>
                {currentSession.status === "FINISHED" ? (
                  <div className="session-log">
                    {(currentSession.results ?? []).map((result) => (
                      <span key={result.playerId}>
                        {result.rank}위 · {result.outcome} · {result.score}점
                      </span>
                    ))}
                  </div>
                ) : (
                  <div className="room-actions">
                    <button
                      className="wide-button"
                      onClick={() =>
                        sendGameAction(currentSession.id, currentRoom?.id, "davinci.pass", setCurrentSession)
                      }
                    >
                      턴 진행
                      <ChevronRight size={18} />
                    </button>
                    <button
                      className="wide-button dark"
                      onClick={() =>
                        sendGameAction(currentSession.id, currentRoom?.id, "davinci.finish", setCurrentSession)
                      }
                    >
                      게임 종료
                      <ChevronRight size={18} />
                    </button>
                  </div>
                )}
              </div>
            </div>
          ) : null}

          <div className="status-strip">
            <span>
              <ShieldCheck size={17} />
              재접속 60초 보장
            </span>
            <span>
              <MessageCircle size={17} />
              로비 채팅 활성
            </span>
          </div>
        </aside>

        <section className="games-panel" aria-label="추천 게임">
          <div className="panel-heading">
            <div>
              <span className="eyebrow compact">
                <Sparkles size={15} />
                인원 기반 추천
              </span>
              <h2>{activePlayerCount}명이 바로 플레이 가능한 게임</h2>
            </div>
            <label className="search-box">
              <Search size={17} aria-hidden="true" />
              <input type="search" placeholder="게임 검색" aria-label="게임 검색" />
            </label>
          </div>

          <div className="filter-row" aria-label="카테고리 필터">
            {["전체", "전략", "블러핑", "순발력", "숫자/조합"].map((category) => (
              <button className={category === "전체" ? "active" : ""} key={category}>
                {category}
              </button>
            ))}
          </div>

          <div className="game-grid">
            {recommendedGames.map((game) => (
              <article
                className="game-card"
                key={game.id}
                style={{ "--accent": game.accent } as CSSProperties}
              >
                <div className="game-art" aria-hidden="true">
                  <span className="token one" />
                  <span className="token two" />
                  <span className="mini-card" />
                </div>
                <div className="game-card-body">
                  <div>
                    <h3>{game.title}</h3>
                    <p>{game.categories.join(" · ")}</p>
                  </div>
                  <dl className="game-meta">
                    <div>
                      <UsersRound size={15} />
                      <dt>인원</dt>
                      <dd>{game.players}</dd>
                    </div>
                    <div>
                      <Clock3 size={15} />
                      <dt>시간</dt>
                      <dd>{game.time}</dd>
                    </div>
                    <div>
                      <Trophy size={15} />
                      <dt>난이도</dt>
                      <dd>{game.difficulty}</dd>
                    </div>
                  </dl>
                </div>
                <button className="game-action" aria-label={`${game.title} 선택`}>
                  선택
                  <ChevronRight size={17} />
                </button>
              </article>
            ))}
          </div>
        </section>
      </section>

      <section className="dock" aria-label="모바일 빠른 실행">
        <button>
          <Gamepad2 size={20} />
          로비
        </button>
        <button>
          <Bot size={20} />
          튜토리얼
        </button>
        <button className="dock-primary">
          <Play size={20} />
          시작
        </button>
        <button>
          <Crown size={20} />
          랭킹
        </button>
      </section>
    </main>
  );
}

function demoParticipants(): RoomParticipant[] {
  return ["Guest_8391", "Nara", "Min", "Seo"].map((nickname, index) => ({
    user: { id: `demo-${nickname}`, nickname },
    ready: index > 0,
    host: index === 0,
    seatIndex: index
  }));
}

function readGuestSession(): GuestSession | null {
  try {
    const raw = localStorage.getItem(guestStorageKey);
    return raw ? (JSON.parse(raw) as GuestSession) : null;
  } catch {
    return null;
  }
}

function saveGuestSession(session: GuestSession) {
  localStorage.setItem(guestStorageKey, JSON.stringify(session));
}

async function createGuest(apiURL: string): Promise<GuestSession> {
  const response = await fetch(`${apiURL}/api/guests`, { method: "POST" });
  if (!response.ok) throw new Error("failed to create guest");

  const session = (await response.json()) as GuestSession;
  saveGuestSession(session);
  return session;
}

async function createRoom(
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readGuestSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>("/api/rooms", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ gameId: "davinci", maxPlayers: 4 })
    });
    onRoom(data.room);
    onMessage(`${data.room.code} 코드를 친구에게 공유하세요.`);
  } catch {
    onMessage("방 생성에 실패했습니다.");
  }
}

async function joinRoom(
  code: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readGuestSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>("/api/rooms/join", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ code })
    });
    onRoom(data.room);
    onMessage(`${data.room.code} 방에 입장했습니다.`);
  } catch {
    onMessage("방 코드를 확인해주세요.");
  }
}

async function setReady(
  roomID: string,
  ready: boolean,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readGuestSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/ready`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ ready })
    });
    onRoom(data.room);
    onMessage(ready ? "준비 완료했습니다." : "준비를 취소했습니다.");
  } catch {
    onMessage("Ready 변경에 실패했습니다.");
  }
}

async function startGame(
  roomID: string,
  onRoom: (room: Room) => void,
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void
) {
  const session = readGuestSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room; session: GameSession }>(
      `/api/rooms/${roomID}/start`,
      session.sessionToken,
      { method: "POST" }
    );
    onRoom(data.room);
    onSession(data.session);
    onMessage("게임을 시작했습니다.");
  } catch {
    onMessage("모든 참가자가 Ready 상태인지 확인해주세요.");
  }
}

async function sendGameAction(
  sessionID: string,
  roomID: string | undefined,
  type: "davinci.pass" | "davinci.finish",
  onSession: (session: GameSession) => void
) {
  const guest = readGuestSession();
  if (!guest || !roomID) return;

  const data = await authorizedJSON<{ session: GameSession }>(
    `/api/sessions/${sessionID}/actions`,
    guest.sessionToken,
    {
      method: "POST",
      body: JSON.stringify({
        roomId: roomID,
        type,
        clientRequestId: crypto.randomUUID()
      })
    }
  );
  onSession(data.session);
}

async function authorizedJSON<T>(
  path: string,
  token: string,
  init: RequestInit
): Promise<T> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...init.headers
    }
  });

  if (!response.ok) throw new Error("request failed");
  return (await response.json()) as T;
}
