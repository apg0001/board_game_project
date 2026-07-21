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
  cover: string;
  mark: string;
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
    accent: "#c95f42",
    cover: "DALMUTI",
    mark: "1"
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
    accent: "#6f5fbf",
    cover: "WEREWOLF",
    mark: "夜"
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
    accent: "#2f78b7",
    cover: "RUMMIKUB",
    mark: "30"
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
    accent: "#a3682a",
    cover: "BANG!",
    mark: "!"
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
    accent: "#1c8c8c",
    cover: "DAVINCI",
    mark: "?"
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
    accent: "#d24f6a",
    cover: "HALLI",
    mark: "5"
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
    accent: "#2f8e74",
    cover: "SPLENDOR",
    mark: "15"
  },
  {
    id: "sutda",
    title: "섯다",
    players: "2-10명",
    minPlayers: 2,
    maxPlayers: 10,
    time: "8분",
    difficulty: "보통",
    categories: ["블러핑", "카드"],
    accent: "#8f3f71",
    cover: "SUTDA",
    mark: "38"
  },
  {
    id: "gostop",
    title: "고스톱",
    players: "2-3명",
    minPlayers: 2,
    maxPlayers: 3,
    time: "20분",
    difficulty: "보통",
    categories: ["전략", "카드"],
    accent: "#b28a22",
    cover: "GO-STOP",
    mark: "光"
  },
  {
    id: "onecard",
    title: "원카드",
    players: "2-6명",
    minPlayers: 2,
    maxPlayers: 6,
    time: "10분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#2c7a9b",
    cover: "ONE CARD",
    mark: "A"
  },
  {
    id: "jokerdraw",
    title: "조커뽑기",
    players: "2-8명",
    minPlayers: 2,
    maxPlayers: 8,
    time: "8분",
    difficulty: "쉬움",
    categories: ["카드", "파티"],
    accent: "#7a4fb0",
    cover: "JOKER",
    mark: "J"
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

interface AuthSession {
  sessionToken: string;
  user: {
    id: string;
    username: string;
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

interface RoomSpectator {
  user: {
    id: string;
    nickname: string;
  };
}

interface Room {
  id: string;
  code: string;
  gameId: string;
  status: "LOBBY" | "PLAYING" | "FINISHED" | "CLOSED";
  activeSessionId?: string;
  playingPlayerIds?: string[];
  maxPlayers: number;
  spectators: RoomSpectator[];
  options: {
    turnSeconds: number;
  };
  participants: RoomParticipant[];
}

interface DavinciTile {
  color: "black" | "white" | "hidden";
  value: number;
  revealed: boolean;
}

interface DavinciPlayer {
  playerId: string;
  tiles?: DavinciTile[];
  deck?: unknown[];
  faceUp?: Array<{
    fruit: string;
    count: number;
  }>;
  score?: number;
  tokens?: Record<string, number>;
  bonuses?: Record<string, number>;
  cards?: SplendorCard[];
  hand?: HandCard[];
  handSize?: number;
  rack?: RummikubTile[];
  rackSize?: number;
  initialMelded?: boolean;
  role?: string;
  hp?: number;
  maxHp?: number;
  alive?: boolean;
  drawn?: boolean;
  bangUsed?: boolean;
  rankName?: string;
  folded?: boolean;
  ready?: boolean;
  captured?: HandCard[];
  goCount?: number;
  passed?: boolean;
  out?: boolean;
  originalRole?: string;
  currentRole?: string;
  seenRoles?: Record<string, string>;
  votedFor?: string;
  active: boolean;
}

interface HandCard {
  id: string;
  rank?: number | string;
  suit?: string;
  value?: number;
  type?: string;
  month?: number;
  gwang?: boolean;
  kind?: string;
  joker?: boolean;
}

interface RummikubTile {
  id: string;
  color: string;
  number: number;
  joker: boolean;
}

interface SplendorCard {
  id: string;
  color: string;
  points: number;
  cost: Record<string, number>;
}

interface GameSession {
  id: string;
  roomId: string;
  gameId: string;
  status: "ACTIVE" | "FINISHED" | "ABORTED";
  state: {
    currentPlayerIndex?: number;
    turnIndex?: number;
    round?: number;
    phase?: "NIGHT" | "DISCUSSION" | "FINISHED";
    log?: string[];
    finished?: boolean;
    players?: DavinciPlayer[];
    bank?: Record<string, number>;
    market?: SplendorCard[];
    currentTrick?: {
      rank?: number;
      count?: number;
      playerId?: string;
    };
    finishOrder?: string[];
    center?: string[];
    executed?: string[];
    winningTeam?: string;
    table?: RummikubTile[][];
    pool?: RummikubTile[];
    winner?: string;
    pot?: number;
    winnerId?: string;
    loserId?: string;
    field?: HandCard[];
    discardPile?: HandCard[];
    drawPile?: HandCard[];
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
    message?: ChatMessage;
    presence?: Presence;
  };
}

interface ChatMessage {
  id: string;
  roomId: string;
  user: {
    id: string;
    nickname: string;
  };
  text: string;
  kind: "chat" | "emoji";
  createdAt: string;
}

interface Presence {
  userId: string;
  roomId: string;
  sessionId?: string;
  status: "ONLINE" | "DISCONNECTED";
  expiresAt?: string;
}

interface LeaderboardRow {
  userId: string;
  gameId: string;
  wins: number;
  losses: number;
  draws: number;
  playCount: number;
  mmr: number;
}

interface TutorialGuide {
  gameId: string;
  title: string;
  summary: string;
  tips: string[];
  steps: Array<{
    title: string;
    description: string;
    actionHint: string;
  }>;
}

const guestStorageKey = "board-table.guest-session";
const authStorageKey = "board-table.auth-session";
const roomStorageKey = "board-table.current-room-id";
const sessionStorageKey = "board-table.current-session-id";

export function App() {
  const [apiGames, setApiGames] = useState<ApiGame[]>([]);
  const [serverStatus, setServerStatus] = useState<"연결됨" | "오프라인 모드">("오프라인 모드");
  const [guestSession, setGuestSession] = useState<GuestSession | null>(() => readGuestSession());
  const [authSession, setAuthSession] = useState<AuthSession | null>(() => readAuthSession());
  const [authUsername, setAuthUsername] = useState("");
  const [authPassword, setAuthPassword] = useState("");
  const [authNickname, setAuthNickname] = useState("");
  const [currentRoom, setCurrentRoom] = useState<Room | null>(null);
  const [currentSession, setCurrentSession] = useState<GameSession | null>(null);
  const [guessTarget, setGuessTarget] = useState("");
  const [guessTileIndex, setGuessTileIndex] = useState(0);
  const [guessColor, setGuessColor] = useState<"black" | "white">("black");
  const [guessValue, setGuessValue] = useState(0);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [presenceByUser, setPresenceByUser] = useState<Record<string, Presence>>({});
  const [leaderboard, setLeaderboard] = useState<LeaderboardRow[]>([]);
  const [selectedGameId, setSelectedGameId] = useState("davinci");
  const [tutorialGuide, setTutorialGuide] = useState<TutorialGuide | null>(null);
  const [gameSearch, setGameSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("전체");
  const [turnSeconds, setTurnSeconds] = useState(60);
  const [roomCodeInput, setRoomCodeInput] = useState("");
  const [roomMessage, setRoomMessage] = useState("방을 만들거나 초대 코드를 입력하세요.");
  const [selectedTileIds, setSelectedTileIds] = useState<string[]>([]);
  const [selectedPlayerIds, setSelectedPlayerIds] = useState<string[]>([]);
  const [authDialogOpen, setAuthDialogOpen] = useState(false);
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const playerToken = authSession?.sessionToken ?? guestSession?.sessionToken ?? "";
  const playerID = authSession?.user.id ?? guestSession?.user.id ?? "";
  const playerNickname = authSession?.user.nickname ?? guestSession?.user.nickname ?? "";

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
    if (!authSession?.sessionToken) return;
    const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
    fetch(`${apiURL}/api/auth/me`, {
      headers: { Authorization: `Bearer ${authSession.sessionToken}` }
    })
      .then((response) => {
        if (!response.ok) throw new Error("auth session expired");
        return response.json() as Promise<{ user: AuthSession["user"] }>;
      })
      .then((data) => {
        setAuthSession((current) => {
          if (!current || current.user.id === data.user.id) return current;
          const refreshed = { ...current, user: data.user };
          saveAuthSession(refreshed);
          return refreshed;
        });
      })
      .catch(() => {
        localStorage.removeItem(authStorageKey);
        setAuthSession(null);
        setRoomMessage("로그인 세션이 만료되어 게스트로 전환했습니다.");
      });
  }, [authSession?.sessionToken]);

  useEffect(() => {
    if (!playerToken) return;
    authorizedJSON<{ presence: Presence }>("/api/reconnect", playerToken, { method: "POST" })
      .then((data) => {
        setPresenceByUser((previous) => ({ ...previous, [data.presence.userId]: data.presence }));
      })
      .catch(() => undefined);
  }, [playerToken]);

  useEffect(() => {
    if (!playerToken) return;

    const roomID = localStorage.getItem(roomStorageKey);
    const sessionID = localStorage.getItem(sessionStorageKey);

    if (roomID) {
      fetchRoom(roomID)
        .then((room) => {
          setCurrentRoom(normalizeRoom(room));
          setSelectedGameId(room.gameId);
          setRoomMessage(`${room.code} 방을 복구했습니다.`);
        })
        .catch(() => localStorage.removeItem(roomStorageKey));
    }

    if (sessionID) {
      fetchSession(sessionID, playerToken)
        .then((session) => {
          setCurrentSession(session);
          setSelectedGameId(session.gameId);
        })
        .catch(() => localStorage.removeItem(sessionStorageKey));
    }
  }, [playerToken]);

  useEffect(() => {
    if (currentRoom) {
      localStorage.setItem(roomStorageKey, currentRoom.id);
      setSelectedGameId(currentRoom.gameId);
      return;
    }
    localStorage.removeItem(roomStorageKey);
  }, [currentRoom]);

  useEffect(() => {
    if (currentSession) {
      localStorage.setItem(sessionStorageKey, currentSession.id);
      return;
    }
    localStorage.removeItem(sessionStorageKey);
  }, [currentSession]);

  useEffect(() => {
    if (!currentRoom || !playerID) return;

    const wsURL = import.meta.env.VITE_WS_URL ?? "ws://localhost:4000/ws";
    const socket = new WebSocket(
      `${wsURL}?room=${encodeURIComponent(`room:${currentRoom.id}`)}&user=${encodeURIComponent(
        playerID
      )}`
    );

    socket.onopen = () => {
      setRoomMessage(`${currentRoom.code} 방에 실시간으로 연결되었습니다.`);
    };

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as RealtimeMessage;
      if (message.type === "room.updated") {
        const nextRoom = message.payload.room;
        if (!nextRoom) return;
        setCurrentRoom(normalizeRoom(nextRoom));
        setRoomMessage(`${nextRoom.code} 방 상태가 갱신되었습니다.`);
      }
      if (message.type === "chat.message" && message.payload.message) {
        setChatMessages((previous) => [...previous.slice(-49), message.payload.message!]);
      }
      if (message.type === "presence.updated" && message.payload.presence) {
        setPresenceByUser((previous) => ({
          ...previous,
          [message.payload.presence!.userId]: message.payload.presence!
        }));
      }
    };

    socket.onclose = () => {
      setRoomMessage("방 실시간 연결이 끊겼습니다. API 상태는 유지됩니다.");
    };

    return () => {
      socket.close();
    };
  }, [currentRoom?.id, playerID]);

  useEffect(() => {
    if (!currentRoom || !playerToken) return;

    fetchRoomChat(currentRoom.id)
      .then(setChatMessages)
      .catch(() => undefined);

    markPresence(currentRoom.id, currentSession?.id, "ONLINE").catch(() => undefined);

    const markDisconnected = () => {
      const token = readPlayerSession()?.sessionToken;
      if (!token) return;
      const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
      const body = JSON.stringify({
        status: "DISCONNECTED",
        sessionId: currentSession?.id
      });
      fetch(`${apiURL}/api/rooms/${currentRoom.id}/presence`, {
        method: "POST",
        keepalive: true,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`
        },
        body
      }).catch(() => undefined);
    };
    window.addEventListener("beforeunload", markDisconnected);
    return () => window.removeEventListener("beforeunload", markDisconnected);
  }, [currentRoom?.id, currentSession?.id, playerToken]);

  useEffect(() => {
    fetchLeaderboard(selectedGameId).then(setLeaderboard).catch(() => undefined);
  }, [currentSession?.status, selectedGameId]);

  useEffect(() => {
    fetchTutorial(selectedGameId).then(setTutorialGuide).catch(() => setTutorialGuide(null));
  }, [selectedGameId]);

  useEffect(() => {
    if (!currentRoom) {
      setSelectedPlayerIds([]);
      return;
    }
    const game = games.find((item) => item.id === currentRoom.gameId) ?? games[4];
    const participantIDs = currentRoom.participants.map((participant) => participant.user.id);
    setSelectedPlayerIds((current) => {
      const valid = current.filter((id) => participantIDs.includes(id));
      if (valid.length > 0) return valid.slice(0, game.maxPlayers);
      return participantIDs.slice(0, game.maxPlayers);
    });
  }, [currentRoom?.id, currentRoom?.gameId, currentRoom?.participants]);

  useEffect(() => {
    if (!currentRoom?.activeSessionId || !playerToken) return;
    if (currentSession?.id === currentRoom.activeSessionId) return;

    fetchSession(currentRoom.activeSessionId, playerToken)
      .then(setCurrentSession)
      .catch(() => undefined);
  }, [currentRoom?.activeSessionId, currentSession?.id, playerToken]);

  useEffect(() => {
    if (!currentSession || !playerID) return;

    const wsURL = import.meta.env.VITE_WS_URL ?? "ws://localhost:4000/ws";
    const socket = new WebSocket(
      `${wsURL}?room=${encodeURIComponent(`game:${currentSession.id}`)}&user=${encodeURIComponent(
        playerID
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
  }, [currentSession?.id, playerID]);

  const recommendedGames = useMemo(() => {
    const source = apiGames.length === 0 ? fallbackRecommendedGames : apiGames.map((apiGame) => {
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
        accent: fallback?.accent ?? "#2f8e74",
        cover: fallback?.cover ?? apiGame.title,
        mark: fallback?.mark ?? apiGame.title.slice(0, 1)
      };
    });

    return source
      .filter((game) => game.title.includes(gameSearch.trim()) || gameSearch.trim() === "")
      .filter((game) => categoryFilter === "전체" || game.categories.includes(categoryFilter as GameCategory));
  }, [apiGames, categoryFilter, gameSearch]);

  const activePlayerCount = currentRoom?.participants.length ?? playerCount;
  const partyMembers = currentRoom?.participants ?? [];
  const me = currentRoom?.participants.find(
    (participant) => participant.user.id === playerID
  );
  const davinciPlayers = currentSession?.state.players ?? [];
  const myDavinciPlayer = davinciPlayers.find((player) => player.playerId === playerID);
  const opponentPlayers = davinciPlayers.filter((player) => player.playerId !== playerID);
  const targetPlayer = opponentPlayers.find((player) => player.playerId === guessTarget) ?? opponentPlayers[0];
  const currentTurnPlayer = davinciPlayers[currentSession?.state.currentPlayerIndex ?? 0];
  const isMyTurn = currentTurnPlayer?.playerId === playerID;
  const selectedGame = games.find((game) => game.id === selectedGameId) ?? games[4];
  const selectedRoomGame = games.find((game) => game.id === currentRoom?.gameId) ?? selectedGame;
  const selectedPlayerSet = new Set(selectedPlayerIds);
  const selectedPlayers = partyMembers.filter((participant) => selectedPlayerSet.has(participant.user.id));
  const selectedReady = selectedPlayers.length > 0 && selectedPlayers.every((participant) => participant.ready);
  const selectedCountValid =
    selectedPlayers.length >= selectedRoomGame.minPlayers && selectedPlayers.length <= selectedRoomGame.maxPlayers;
  const readyCount = partyMembers.filter((participant) => participant.ready).length;
  const amHost = Boolean(me?.host);
  const canStartGame = Boolean(currentRoom && amHost && selectedReady && selectedCountValid && currentRoom.status === "LOBBY");
  const currentTurnName = currentTurnPlayer ? participantName(currentRoom, currentTurnPlayer.playerId) : "대기 중";
  const splendorColors = ["white", "blue", "green", "red", "black"];
  const splendorMe = davinciPlayers.find((player) => player.playerId === playerID);
  const dalmutiMe = davinciPlayers.find((player) => player.playerId === playerID);
  const dalmutiGroups = groupDalmutiHand(dalmutiMe?.hand ?? []);
  const werewolfMe = davinciPlayers.find((player) => player.playerId === playerID);
  const werewolfOthers = davinciPlayers.filter((player) => player.playerId !== playerID);
  const rummikubMe = davinciPlayers.find((player) => player.playerId === playerID);
  const bangMe = davinciPlayers.find((player) => player.playerId === playerID);
  const bangTargets = davinciPlayers.filter((player) => player.playerId !== playerID && player.alive !== false);
  const sutdaMe = davinciPlayers.find((player) => player.playerId === playerID);
  const gostopMe = davinciPlayers.find((player) => player.playerId === playerID);
  const onecardMe = davinciPlayers.find((player) => player.playerId === playerID);
  const onecardTopCard = currentSession?.state.discardPile?.[(currentSession.state.discardPile?.length ?? 0) - 1];
  const jokerdrawMe = davinciPlayers.find((player) => player.playerId === playerID);
  const jokerdrawTargets = davinciPlayers.filter(
    (player) => player.playerId !== playerID && player.active && !player.out
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
              {playerNickname ? playerNickname.slice(-2) : "G"}
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
              <button className="primary-button" onClick={() => quickMatch(selectedGameId, setCurrentRoom, setRoomMessage)}>
                <Play size={18} />
                퀵매치
              </button>
              {currentRoom && amHost ? (
                <button
                  className="secondary-button"
                  onClick={() => cancelQuickMatch(currentRoom.id, currentRoom.gameId, setRoomMessage)}
                >
                  <Search size={18} />
                  매칭 취소
                </button>
              ) : null}
              <button className="secondary-button" onClick={() => createRoom(selectedGameId, setCurrentRoom, setRoomMessage)}>
                <Plus size={18} />
                초대 방
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
                <span>{playerNickname || "게스트 준비 중"}</span>
              <h2>{activePlayerCount}명 입장 중 · Ready {readyCount}/{partyMembers.length || activePlayerCount}</h2>
              </div>
              <UsersRound size={22} />
            </div>
            <div className="party-list">
              {(partyMembers.length > 0 ? partyMembers : demoParticipants()).map((participant) => (
                <div className="party-member" key={participant.user.id}>
                  <span>{participant.user.nickname.slice(0, 1)}</span>
                  <div>
                    <strong>{participant.user.nickname}</strong>
                    <small>
                      {participant.host ? "방장" : participant.ready ? "준비 완료" : "대기 중"}
                      {" · "}
                      {presenceByUser[participant.user.id]?.status === "DISCONNECTED" ? "재접속 대기" : "온라인"}
                      {currentRoom?.status === "PLAYING" && !currentRoom.playingPlayerIds?.includes(participant.user.id)
                        ? " · 이번 판 관전"
                        : ""}
                    </small>
                    {currentRoom && amHost && participant.user.id !== playerID ? (
                      <div className="member-actions">
                        <button
                          onClick={() =>
                            transferHost(currentRoom.id, participant.user.id, setCurrentRoom, setRoomMessage)
                          }
                        >
                          위임
                        </button>
                        <button
                          onClick={() =>
                            kickPlayer(currentRoom.id, participant.user.id, setCurrentRoom, setRoomMessage)
                          }
                        >
                          강퇴
                        </button>
                      </div>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
            {currentRoom?.spectators?.length ? (
              <div className="spectator-list">
                <span>관전 {currentRoom.spectators.length}명</span>
                {currentRoom.spectators.map((spectator) => (
                  <small key={spectator.user.id}>{spectator.user.nickname}</small>
                ))}
              </div>
            ) : null}
          </div>

          <div className="room-card room-setup-card">
            <div className="section-title">
              <div>
                <span>방 코드</span>
                <h2>{currentRoom?.code ?? "없음"}</h2>
                <small className="status-badge">{currentRoom?.status ?? "HOME"}</small>
              </div>
              <LockKeyhole size={22} />
            </div>
            <div className="room-actions">
              <div className="selected-game-strip">
                <span>선택 게임</span>
                <strong>{selectedGame.title}</strong>
                <small>
                  게임 {selectedGame.players} · 초대 로비 최대 12명
                </small>
              </div>
              <p className="room-helper">
                게임 최대 인원을 넘으면 방장이 이번 판 플레이어를 고르고 나머지는 같은 방에서 관전합니다.
              </p>
              <button className="wide-button" onClick={() => createRoom(selectedGameId, setCurrentRoom, setRoomMessage)}>
                {selectedGame.title} 방 만들기
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
                  className="wide-button"
                  onClick={() => {
                    navigator.clipboard?.writeText(currentRoom.code).catch(() => undefined);
                    setRoomMessage(`${currentRoom.code} 코드를 복사했습니다.`);
                  }}
                >
                  코드 복사
                  <ChevronRight size={18} />
                </button>
              ) : null}
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
                <div className="option-row">
                  <label>
                    턴 제한
                    <input
                      type="number"
                      min="10"
                      max="300"
                      value={turnSeconds}
                      onChange={(event) => setTurnSeconds(Number(event.target.value))}
                    />
                  </label>
                  <button
                    onClick={() => updateRoomOptions(currentRoom.id, turnSeconds, setCurrentRoom, setRoomMessage)}
                    disabled={!amHost}
                  >
                    적용
                  </button>
                </div>
              ) : null}
              {currentRoom ? (
                <button
                  className="wide-button"
                  onClick={() => spectateRoom(currentRoom.id, setCurrentRoom, setRoomMessage)}
                >
                  관전 입장
                  <ChevronRight size={18} />
                </button>
              ) : null}
              {currentRoom ? (
                <div className="player-selection-panel">
                  <div>
                    <strong>이번 판 플레이어</strong>
                    <span>
                      {selectedPlayers.length}/{selectedRoomGame.maxPlayers}명 선택 · 최소 {selectedRoomGame.minPlayers}명
                    </span>
                  </div>
                  <div className="player-selection-grid">
                    {partyMembers.map((participant) => {
                      const checked = selectedPlayerIds.includes(participant.user.id);
                      const locked = !checked && selectedPlayerIds.length >= selectedRoomGame.maxPlayers;
                      return (
                        <label className="player-select-chip" key={participant.user.id}>
                          <input
                            type="checkbox"
                            checked={checked}
                            disabled={!amHost || currentRoom.status !== "LOBBY" || locked}
                            onChange={(event) =>
                              setSelectedPlayerIds((current) =>
                                event.target.checked
                                  ? [...current, participant.user.id].slice(0, selectedRoomGame.maxPlayers)
                                  : current.filter((id) => id !== participant.user.id)
                              )
                            }
                          />
                          <span>{participant.user.nickname}</span>
                          <small>{participant.ready ? "Ready" : "대기"}</small>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ) : null}
              {currentRoom ? (
                <button
                  className="wide-button play-now"
                  onClick={() =>
                    startGame(currentRoom.id, selectedPlayerIds, setCurrentRoom, setCurrentSession, setRoomMessage)
                  }
                  disabled={!canStartGame}
                  title={canStartGame ? "게임 시작" : "방장이 이번 판 플레이어를 고르고 선택된 플레이어가 Ready여야 시작할 수 있습니다."}
                >
                  게임 시작
                  <ChevronRight size={18} />
                </button>
              ) : null}
              {currentRoom ? (
                <button
                  className="wide-button dark"
                  onClick={() =>
                    roomPostAction(
                      currentRoom.id,
                      "leave",
                      setCurrentRoom,
                      () => setCurrentSession(null),
                      setRoomMessage
                    )
                  }
                >
                  방 나가기
                  <ChevronRight size={18} />
                </button>
              ) : null}
            </div>
            <p className="room-message">{roomMessage}</p>
            {currentRoom ? (
              <p className="room-message">
                {currentRoom.gameId} · 로비 {currentRoom.participants.length}/{currentRoom.maxPlayers}명 · 이번 판{" "}
                {selectedPlayers.length}/{selectedRoomGame.maxPlayers}명 · 관전{" "}
                {currentRoom.spectators?.length ?? 0}명 · 턴 {currentRoom.options?.turnSeconds ?? 60}초
              </p>
            ) : null}
            {currentRoom ? (
              <p className="room-message">
                시작 조건: {selectedRoomGame.minPlayers}-{selectedRoomGame.maxPlayers}명 선택 · 선택 플레이어 Ready · 방장 시작
              </p>
            ) : null}
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
                <p>라운드 {currentSession.state.round ?? 1} · 현재 턴 {currentTurnName}</p>
                <div className="session-log">
                  {(currentSession.state.log ?? []).slice(-3).map((item) => (
                    <span key={item}>{item}</span>
                  ))}
                </div>
                {currentSession.status === "FINISHED" ? (
                  <div className="post-game-panel">
                    <div className="session-log">
                      {(currentSession.results ?? []).map((result) => (
                        <span key={result.playerId}>
                          {result.rank}위 · {result.outcome} · {result.score}점
                        </span>
                      ))}
                    </div>
                    <div className="room-actions">
                      <button
                        className="wide-button play-now"
                        onClick={() =>
                          roomPostAction(
                            currentRoom?.id,
                            "rematch",
                            setCurrentRoom,
                            () => setCurrentSession(null),
                            setRoomMessage
                          )
                        }
                      >
                        다시 하기
                        <ChevronRight size={18} />
                      </button>
                      <button
                        className="wide-button"
                        onClick={() =>
                          roomPostAction(
                            currentRoom?.id,
                            "return-lobby",
                            setCurrentRoom,
                            () => setCurrentSession(null),
                            setRoomMessage
                          )
                        }
                      >
                        로비로 가기
                        <ChevronRight size={18} />
                      </button>
                      <button
                        className="wide-button dark"
                        onClick={() => {
                          setCurrentRoom(null);
                          setCurrentSession(null);
                          setRoomMessage("홈으로 돌아왔습니다.");
                        }}
                      >
                        홈으로 가기
                        <ChevronRight size={18} />
                      </button>
                      <button
                        className="wide-button dark"
                        onClick={() =>
                          roomPostAction(
                            currentRoom?.id,
                            "leave",
                            setCurrentRoom,
                            () => setCurrentSession(null),
                            setRoomMessage
                          )
                        }
                      >
                        방 나가기
                        <ChevronRight size={18} />
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="room-actions">
                    {currentSession.gameId === "halli-galli" ? (
                      <div className="room-actions">
                        <div className="halli-board">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>{player.score ?? 0}점 · 덱 {player.deck?.length ?? 0}장</span>
                              <div className="halli-cards">
                                {(player.faceUp ?? []).slice(-3).map((card, index) => (
                                  <span key={`${player.playerId}-${card.fruit}-${card.count}-${index}`}>
                                    {fruitLabel(card.fruit)} {card.count}
                                  </span>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "halli-galli.flip",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                        >
                          카드 펼치기
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button play-now"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "halli-galli.ring",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                        >
                          종 치기
                          <ChevronRight size={18} />
                        </button>
                      </div>
                    ) : currentSession.gameId === "splendor" ? (
                      <div className="room-actions">
                        <div className="splendor-bank" aria-label="보석 은행">
                          {splendorColors.map((color) => (
                            <button
                              className={`gem-button ${color}`}
                              key={color}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "splendor.take_token",
                                  setCurrentSession,
                                  setRoomMessage,
                                  { color }
                                )
                              }
                              disabled={!isMyTurn || (currentSession.state.bank?.[color] ?? 0) <= 0}
                            >
                              <span>{gemLabel(color)}</span>
                              <strong>{currentSession.state.bank?.[color] ?? 0}</strong>
                            </button>
                          ))}
                        </div>
                        <div className="splendor-market" aria-label="시장 카드">
                          {(currentSession.state.market ?? []).map((card, index) => (
                            <button
                              className={`splendor-card ${card.color}`}
                              key={card.id}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "splendor.buy_card",
                                  setCurrentSession,
                                  setRoomMessage,
                                  { marketIndex: index }
                                )
                              }
                              disabled={!isMyTurn}
                            >
                              <strong>{card.points}점</strong>
                              <span>{gemLabel(card.color)} 보너스</span>
                              <small>{formatCost(card.cost)}</small>
                            </button>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                {player.score ?? 0}점 · 카드 {player.cards?.length ?? 0}장
                              </span>
                              <div className="halli-cards">
                                {splendorColors.map((color) => (
                                  <span key={`${player.playerId}-${color}`}>
                                    {gemLabel(color)} {player.tokens?.[color] ?? 0}/{player.bonuses?.[color] ?? 0}
                                  </span>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                        <p className="helper-copy">
                          내 보석 {formatCost(splendorMe?.tokens ?? {})} · 보너스 {formatCost(splendorMe?.bonuses ?? {})}
                        </p>
                      </div>
                    ) : currentSession.gameId === "dalmuti" ? (
                      <div className="room-actions">
                        <div className="dalmuti-status">
                          <strong>
                            현재 트릭{" "}
                            {currentSession.state.currentTrick?.count
                              ? `${dalmutiRankLabel(currentSession.state.currentTrick.rank ?? 0)} ${currentSession.state.currentTrick.count}장`
                              : "리드 대기"}
                          </strong>
                          <span>
                            리더 {participantName(currentRoom, currentSession.state.currentTrick?.playerId ?? "")}
                          </span>
                        </div>
                        <div className="dalmuti-hand" aria-label="내 카드">
                          {dalmutiGroups.map((group) => (
                            <div className="dalmuti-group" key={group.rank}>
                              <strong>{dalmutiRankLabel(group.rank)}</strong>
                              <span>{group.count}장</span>
                              <div className="dalmuti-counts">
                                {Array.from({ length: group.count }, (_, index) => index + 1).map((count) => (
                                  <button
                                    key={`${group.rank}-${count}`}
                                    onClick={() =>
                                      sendGameAction(
                                        currentSession.id,
                                        currentRoom?.id,
                                        "dalmuti.play",
                                        setCurrentSession,
                                        setRoomMessage,
                                        { rank: group.rank, count }
                                      )
                                    }
                                    disabled={!isMyTurn}
                                  >
                                    {count}
                                  </button>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                손패 {player.handSize ?? player.hand?.length ?? 0}장 ·{" "}
                                {player.out ? "아웃" : player.passed ? "패스" : "진행 중"}
                              </span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button dark"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "dalmuti.pass",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn || !currentSession.state.currentTrick?.count}
                        >
                          패스
                          <ChevronRight size={18} />
                        </button>
                      </div>
                    ) : currentSession.gameId === "rummikub" ? (
                      <div className="room-actions">
                        <div className="rummikub-rack" aria-label="내 랙">
                          {(rummikubMe?.rack ?? []).map((tile) => (
                            <button
                              className={`rummikub-tile ${tile.color} ${
                                selectedTileIds.includes(tile.id) ? "selected" : ""
                              }`}
                              key={tile.id}
                              onClick={() => setSelectedTileIds((previous) => toggleSelected(previous, tile.id))}
                            >
                              {tile.joker ? "J" : tile.number}
                            </button>
                          ))}
                        </div>
                        <div className="rummikub-table">
                          {(currentSession.state.table ?? []).map((group, groupIndex) => (
                            <div className="tile-row" key={`group-${groupIndex}`}>
                              {group.map((tile) => (
                                <span className={`rummikub-tile ${tile.color}`} key={tile.id}>
                                  {tile.joker ? "J" : tile.number}
                                </span>
                              ))}
                            </div>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                랙 {player.rackSize ?? player.rack?.length ?? 0}개 ·{" "}
                                {player.initialMelded ? "첫 등록 완료" : "첫 등록 전"}
                              </span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "rummikub.meld",
                              (session) => {
                                setCurrentSession(session);
                                setSelectedTileIds([]);
                              },
                              setRoomMessage,
                              { tileIds: selectedTileIds }
                            )
                          }
                          disabled={!isMyTurn || selectedTileIds.length < 3}
                        >
                          선택 조합 등록
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button dark"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "rummikub.draw",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn}
                        >
                          타일 뽑기
                          <ChevronRight size={18} />
                        </button>
                      </div>
                    ) : currentSession.gameId === "bang" ? (
                      <div className="room-actions">
                        <div className="bang-hand" aria-label="내 카드">
                          {(bangMe?.hand ?? []).map((card) => (
                            <button
                              className={`bang-card ${card.type}`}
                              key={card.id}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "bang.play",
                                  setCurrentSession,
                                  setRoomMessage,
                                  {
                                    cardId: card.id,
                                    targetPlayerId: bangTargets[0]?.playerId ?? ""
                                  }
                                )
                              }
                              disabled={!isMyTurn}
                            >
                              {bangCardLabel(card.type ?? "")}
                            </button>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                {player.role ? bangRoleLabel(player.role) : "비공개"} · HP {player.hp ?? 0}/
                                {player.maxHp ?? 0} · 손패 {player.handSize ?? player.hand?.length ?? 0}
                              </span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "bang.draw",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn || Boolean(bangMe?.drawn)}
                        >
                          카드 2장 뽑기
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button dark"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "bang.end_turn",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn}
                        >
                          턴 종료
                          <ChevronRight size={18} />
                        </button>
	                        {currentSession.state.finished ? (
	                          <p className="helper-copy">승리 진영 {bangWinnerLabel(currentSession.state.winner ?? "")}</p>
	                        ) : null}
	                      </div>
                    ) : currentSession.gameId === "sutda" ? (
                      <div className="room-actions">
                        <div className="hwatu-hand" aria-label="내 섯다 패">
                          {(sutdaMe?.hand ?? []).map((card) => (
                            <span className="hwatu-card" key={card.id}>
                              {card.month}월{card.gwang ? " 광" : ""}
                            </span>
                          ))}
                        </div>
                        <div className="werewolf-panel">
                          <strong>{sutdaMe?.rankName ?? "족보 대기"}</strong>
                          <span>판돈 {currentSession.state.pot ?? 0}</span>
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>{player.folded ? "다이" : player.ready ? "콜" : "대기"}</span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "sutda.call",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn}
                        >
                          콜
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button dark"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "sutda.fold",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn}
                        >
                          다이
                          <ChevronRight size={18} />
                        </button>
                      </div>
                    ) : currentSession.gameId === "gostop" ? (
                      <div className="room-actions">
                        <div className="hwatu-hand" aria-label="내 고스톱 패">
                          {(gostopMe?.hand ?? []).map((card) => (
                            <button
                              className={`hwatu-card ${card.kind}`}
                              key={card.id}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "gostop.play",
                                  setCurrentSession,
                                  setRoomMessage,
                                  { cardId: card.id }
                                )
                              }
                              disabled={!isMyTurn}
                            >
                              {card.month}월 {goStopKindLabel(card.kind ?? "")}
                            </button>
                          ))}
                        </div>
                        <div className="hwatu-field">
                          {(currentSession.state.field ?? []).map((card) => (
                            <span className={`hwatu-card ${card.kind}`} key={card.id}>
                              {card.month}월
                            </span>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                {player.score ?? 0}점 · 획득 {player.captured?.length ?? 0}장 · {player.goCount ?? 0}고
                              </span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "gostop.go",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn || (gostopMe?.score ?? 0) < 3}
                        >
                          고
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button play-now"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "gostop.stop",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn || (gostopMe?.score ?? 0) < 3}
                        >
                          스톱
                          <ChevronRight size={18} />
                        </button>
                      </div>
                    ) : currentSession.gameId === "onecard" ? (
                      <div className="room-actions">
                        <div className="werewolf-panel">
                          <strong>{onecardTopCard ? standardCardLabel(onecardTopCard) : "버린 카드 대기"}</strong>
                          <span>
                            더미 {currentSession.state.drawPile?.length ?? 0}장 · 내 손패 {onecardMe?.hand?.length ?? 0}장
                          </span>
                        </div>
                        <div className="standard-hand" aria-label="내 원카드 손패">
                          {(onecardMe?.hand ?? []).map((card) => (
                            <button
                              className={`standard-card ${card.suit ?? ""}`}
                              key={card.id}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "onecard.play",
                                  setCurrentSession,
                                  setRoomMessage,
                                  { cardId: card.id }
                                )
                              }
                              disabled={!isMyTurn}
                            >
                              {standardCardLabel(card)}
                            </button>
                          ))}
                        </div>
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>{player.active ? "플레이 중" : "아웃"} · 손패 {player.handSize ?? player.hand?.length ?? 0}장</span>
                            </div>
                          ))}
                        </div>
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "onecard.draw",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={!isMyTurn}
                        >
                          카드 뽑기
                          <ChevronRight size={18} />
                        </button>
                        {currentSession.state.finished ? (
                          <p className="helper-copy">승자 {participantName(currentRoom, currentSession.state.winnerId ?? "")}</p>
                        ) : null}
                      </div>
                    ) : currentSession.gameId === "jokerdraw" ? (
                      <div className="room-actions">
                        <div className="standard-hand" aria-label="내 조커뽑기 손패">
                          {(jokerdrawMe?.hand ?? []).map((card) => (
                            <span className={`standard-card readonly ${card.suit ?? ""}`} key={card.id}>
                              {standardCardLabel(card)}
                            </span>
                          ))}
                        </div>
                        <div className="joker-targets">
                          {jokerdrawTargets.map((player) => (
                            <div className="halli-player joker-target" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>남은 카드 {player.handSize ?? player.hand?.length ?? 0}장</span>
                              <div className="joker-card-buttons">
                                {Array.from({ length: player.handSize ?? player.hand?.length ?? 0 }, (_, index) => (
                                  <button
                                    className="mini-card-button"
                                    key={`${player.playerId}-${index}`}
                                    onClick={() =>
                                      sendGameAction(
                                        currentSession.id,
                                        currentRoom?.id,
                                        "jokerdraw.draw",
                                        setCurrentSession,
                                        setRoomMessage,
                                        { targetPlayerId: player.playerId, cardIndex: index }
                                      )
                                    }
                                    disabled={!isMyTurn}
                                    aria-label={`${participantName(currentRoom, player.playerId)} ${index + 1}번째 카드 뽑기`}
                                  >
                                    ?
                                  </button>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                        {currentSession.state.finished ? (
                          <p className="helper-copy">조커 보유 패자 {participantName(currentRoom, currentSession.state.loserId ?? "")}</p>
                        ) : null}
                      </div>
                    ) : currentSession.gameId === "werewolf" ? (
                      <div className="room-actions">
                        <div className="werewolf-panel">
                          <strong>{werewolfRoleLabel(werewolfMe?.originalRole ?? "hidden")}</strong>
                          <span>
                            현재 단계 {werewolfPhaseLabel(currentSession.state.phase)} · 현재 역할{" "}
                            {werewolfRoleLabel(werewolfMe?.currentRole ?? "hidden")}
                          </span>
                          <div className="halli-cards">
                            {Object.entries(werewolfMe?.seenRoles ?? {}).map(([key, role]) => (
                              <span key={key}>
                                {key} {werewolfRoleLabel(role)}
                              </span>
                            ))}
                          </div>
                        </div>
                        {currentSession.state.phase === "NIGHT" ? (
                          <div className="werewolf-actions">
                            {werewolfMe?.originalRole === "seer" ? (
                              <>
                                {werewolfOthers.map((player) => (
                                  <button
                                    className="wide-button"
                                    key={player.playerId}
                                    onClick={() =>
                                      sendGameAction(
                                        currentSession.id,
                                        currentRoom?.id,
                                        "werewolf.see_player",
                                        setCurrentSession,
                                        setRoomMessage,
                                        { targetPlayerId: player.playerId }
                                      )
                                    }
                                  >
                                    {participantName(currentRoom, player.playerId)} 확인
                                    <ChevronRight size={18} />
                                  </button>
                                ))}
                                <button
                                  className="wide-button"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "werewolf.see_center",
                                      setCurrentSession,
                                      setRoomMessage,
                                      { centerIndexes: [0, 1] }
                                    )
                                  }
                                >
                                  중앙 2장 확인
                                  <ChevronRight size={18} />
                                </button>
                              </>
                            ) : null}
                            {werewolfMe?.originalRole === "robber"
                              ? werewolfOthers.map((player) => (
                                  <button
                                    className="wide-button"
                                    key={player.playerId}
                                    onClick={() =>
                                      sendGameAction(
                                        currentSession.id,
                                        currentRoom?.id,
                                        "werewolf.rob",
                                        setCurrentSession,
                                        setRoomMessage,
                                        { targetPlayerId: player.playerId }
                                      )
                                    }
                                  >
                                    {participantName(currentRoom, player.playerId)} 강탈
                                    <ChevronRight size={18} />
                                  </button>
                                ))
                              : null}
                            {werewolfMe?.originalRole === "troublemaker" && werewolfOthers.length >= 2 ? (
                              <button
                                className="wide-button"
                                onClick={() =>
                                  sendGameAction(
                                    currentSession.id,
                                    currentRoom?.id,
                                    "werewolf.troublemake",
                                    setCurrentSession,
                                    setRoomMessage,
                                    {
                                      leftPlayerId: werewolfOthers[0].playerId,
                                      rightPlayerId: werewolfOthers[1].playerId
                                    }
                                  )
                                }
                              >
                                앞의 두 명 바꾸기
                                <ChevronRight size={18} />
                              </button>
                            ) : null}
                            {werewolfMe?.originalRole === "drunk" ? (
                              <button
                                className="wide-button"
                                onClick={() =>
                                  sendGameAction(
                                    currentSession.id,
                                    currentRoom?.id,
                                    "werewolf.drunk_swap",
                                    setCurrentSession,
                                    setRoomMessage,
                                    { centerIndexes: [0] }
                                  )
                                }
                              >
                                중앙 1번과 교환
                                <ChevronRight size={18} />
                              </button>
                            ) : null}
                            <button
                              className="wide-button play-now"
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "werewolf.finish_night",
                                  setCurrentSession,
                                  setRoomMessage
                                )
                              }
                            >
                              밤 종료
                              <ChevronRight size={18} />
                            </button>
                          </div>
                        ) : (
                          <div className="werewolf-actions">
                            <div className="splendor-players">
                              {davinciPlayers.map((player) => (
                                <button
                                  className="wide-button"
                                  key={player.playerId}
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "werewolf.vote",
                                      setCurrentSession,
                                      setRoomMessage,
                                      { targetPlayerId: player.playerId }
                                    )
                                  }
                                  disabled={currentSession.state.phase !== "DISCUSSION"}
                                >
                                  {participantName(currentRoom, player.playerId)} 투표
                                  <ChevronRight size={18} />
                                </button>
                              ))}
                            </div>
                            {currentSession.state.finished ? (
                              <p className="helper-copy">
                                처형 {currentSession.state.executed?.map((id) => participantName(currentRoom, id)).join(", ") || "없음"} · 승리팀{" "}
                                {currentSession.state.winningTeam === "village" ? "마을" : "늑대"}
                              </p>
                            ) : null}
                          </div>
                        )}
                      </div>
                    ) : (
                      <>
                    <div className="tile-board" aria-label="내 타일">
                      <span>내 타일</span>
                      <div className="tile-row">
                        {(myDavinciPlayer?.tiles ?? []).map((tile, index) => (
                          <span className={`davinci-tile ${tile.color}`} key={`${tile.color}-${tile.value}-${index}`}>
                            {tile.value}
                          </span>
                        ))}
                      </div>
                    </div>
                    <div className="tile-board" aria-label="상대 타일">
                      <span>상대 타일</span>
                      {opponentPlayers.map((player) => (
                        <div className="opponent-row" key={player.playerId}>
                          <strong>{participantName(currentRoom, player.playerId)}</strong>
                          <div className="tile-row">
                            {(player.tiles ?? []).map((tile, index) => (
                              <button
                                className={`davinci-tile ${tile.color} ${
                                  targetPlayer?.playerId === player.playerId && guessTileIndex === index ? "selected" : ""
                                }`}
                                key={`${player.playerId}-${index}`}
                                onClick={() => {
                                  setGuessTarget(player.playerId);
                                  setGuessTileIndex(index);
                                }}
                              >
                                {tile.value >= 0 ? tile.value : "?"}
                              </button>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                    <div className="guess-controls">
                      <select
                        value={guessColor}
                        onChange={(event) => setGuessColor(event.target.value as "black" | "white")}
                        aria-label="추측 색상"
                      >
                        <option value="black">검정</option>
                        <option value="white">흰색</option>
                      </select>
                      <input
                        type="number"
                        min="0"
                        max="11"
                        value={guessValue}
                        onChange={(event) => setGuessValue(Number(event.target.value))}
                        aria-label="추측 숫자"
                      />
                    </div>
                    <button
                      className="wide-button"
                      onClick={() =>
                        sendGuessAction(
                          currentSession.id,
                          currentRoom?.id,
                          {
                            targetPlayerId: targetPlayer?.playerId ?? "",
                            tileIndex: guessTileIndex,
                            color: guessColor,
                            value: guessValue
                          },
                          setCurrentSession,
                          setRoomMessage
                        )
                      }
                      disabled={!isMyTurn || !targetPlayer}
                    >
                      추측하기
                      <ChevronRight size={18} />
                    </button>
                    <button
                      className="wide-button dark"
                      onClick={() =>
                        sendGameAction(
                          currentSession.id,
                          currentRoom?.id,
                          "davinci.pass",
                          setCurrentSession,
                          setRoomMessage
                        )
                      }
                    >
                      턴 넘기기
                      <ChevronRight size={18} />
                    </button>
                      </>
                    )}
                  </div>
                )}
              </div>
            </div>
          ) : null}

          {currentRoom ? (
            <div className="room-card">
              <div className="section-title">
                <div>
                  <span>채팅</span>
                  <h2>로비 메시지</h2>
                </div>
                <MessageCircle size={22} />
              </div>
              <div className="chat-list">
                {chatMessages.slice(-5).map((message) => (
                  <span key={message.id}>
                    <strong>{message.user.nickname}</strong> {message.text}
                  </span>
                ))}
              </div>
              <div className="chat-actions">
                <input
                  value={chatInput}
                  maxLength={160}
                  onChange={(event) => setChatInput(event.target.value.slice(0, 160))}
                  onKeyDown={(event) => {
                    if (event.key !== "Enter" || !chatInput.trim()) return;
                    sendChat(currentRoom.id, chatInput, "chat").then((message) => {
                      setChatMessages((previous) => [...previous.slice(-49), message]);
                      setChatInput("");
                    });
                  }}
                  placeholder="메시지"
                  aria-label="채팅 메시지"
                />
                <button
                  disabled={!chatInput.trim()}
                  onClick={() => {
                    if (!chatInput.trim()) return;
                    sendChat(currentRoom.id, chatInput, "chat").then((message) => {
                      setChatMessages((previous) => [...previous.slice(-49), message]);
                      setChatInput("");
                    });
                  }}
                >
                  전송
                </button>
              </div>
              <div className="emoji-row">
                {["👍", "🎉", "😮"].map((emoji) => (
                  <button
                    key={emoji}
                    onClick={() =>
                      sendChat(currentRoom.id, emoji, "emoji").then((message) =>
                        setChatMessages((previous) => [...previous.slice(-49), message])
                      )
                    }
                  >
                    {emoji}
                  </button>
                ))}
              </div>
            </div>
          ) : null}

          <div className="room-card">
            <div className="section-title">
              <div>
                <span>계정</span>
                <h2>{authSession ? authSession.user.nickname : "게스트 플레이 중"}</h2>
              </div>
              <Crown size={22} />
            </div>
            {authSession ? (
              <button
                className="wide-button dark"
                onClick={() => {
                  localStorage.removeItem(authStorageKey);
                  setAuthSession(null);
                  setCurrentRoom(null);
                  setCurrentSession(null);
                  setRoomMessage("로그아웃했습니다. 게스트로 계속 플레이할 수 있습니다.");
                }}
              >
                로그아웃
                <ChevronRight size={18} />
              </button>
            ) : (
              <div className="auth-cta">
                <p>전적 저장과 고정 닉네임은 계정으로 사용할 수 있습니다.</p>
                <button
                  className="wide-button"
                  onClick={() => {
                    setAuthMode("login");
                    setAuthDialogOpen(true);
                  }}
                >
                  로그인
                  <ChevronRight size={18} />
                </button>
                <button
                  className="wide-button dark"
                  onClick={() => {
                    setAuthMode("register");
                    setAuthDialogOpen(true);
                  }}
                >
                  회원가입
                  <ChevronRight size={18} />
                </button>
              </div>
            )}
          </div>

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
              <input
                type="search"
                placeholder="게임 검색"
                aria-label="게임 검색"
                value={gameSearch}
                onChange={(event) => setGameSearch(event.target.value)}
              />
            </label>
          </div>

          <div className="filter-row" aria-label="카테고리 필터">
            {["전체", "전략", "블러핑", "순발력", "숫자/조합"].map((category) => (
              <button
                className={category === categoryFilter ? "active" : ""}
                key={category}
                onClick={() => setCategoryFilter(category)}
              >
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
                <div className={`game-art game-art-${game.id}`} aria-hidden="true">
                  <span className="cover-title">{game.cover}</span>
                  <span className="cover-mark">{game.mark}</span>
                  <span className="cover-card cover-card-a" />
                  <span className="cover-card cover-card-b" />
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
                <button
                  className="game-action"
                  aria-label={`${game.title} 선택`}
                  onClick={() => {
                    setSelectedGameId(game.id);
                    setRoomMessage(`${game.title} 선택됨`);
                  }}
                >
                  {selectedGameId === game.id ? "선택됨" : "선택"}
                  <ChevronRight size={17} />
                </button>
              </article>
            ))}
          </div>

          <div className="leaderboard-panel">
            <div className="section-title">
              <div>
                <span>Leaderboard</span>
                <h2>{selectedGame.title} 랭킹</h2>
              </div>
              <Trophy size={22} />
            </div>
            <div className="leaderboard-list">
              {leaderboard.length === 0 ? (
                <span>아직 기록이 없습니다.</span>
              ) : (
                leaderboard.slice(0, 5).map((row, index) => (
                  <span key={row.userId}>
                    {index + 1}. {row.userId} · {row.mmr} MMR · {row.wins}승
                  </span>
                ))
              )}
            </div>
            <button
              className="mini-command"
              onClick={() => fetchLeaderboard(selectedGameId).then(setLeaderboard).catch(() => undefined)}
            >
              랭킹 새로고침
            </button>
          </div>

          {tutorialGuide ? (
            <div className="leaderboard-panel">
              <div className="section-title">
                <div>
                  <span>Tutorial</span>
                  <h2>{tutorialGuide.title}</h2>
                </div>
                <Bot size={22} />
              </div>
              <p className="tutorial-summary">{tutorialGuide.summary}</p>
              <div className="tutorial-tips">
                {tutorialGuide.tips.map((tip) => (
                  <span key={tip}>{tip}</span>
                ))}
              </div>
              <div className="tutorial-steps">
                {tutorialGuide.steps.map((step, index) => (
                  <span key={step.title}>
                    {index + 1}. {step.title} · {step.actionHint}
                  </span>
                ))}
              </div>
            </div>
          ) : null}

          <div className="leaderboard-panel">
            <div className="section-title">
              <div>
                <span>Local</span>
                <h2>세션 관리</h2>
              </div>
              <ShieldCheck size={22} />
            </div>
            <button
              className="mini-command danger"
              onClick={() => {
                localStorage.removeItem(roomStorageKey);
                localStorage.removeItem(sessionStorageKey);
                setCurrentRoom(null);
                setCurrentSession(null);
                setRoomMessage("로컬 방/게임 복구 정보를 초기화했습니다.");
              }}
            >
              로컬 세션 초기화
            </button>
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
      {authDialogOpen ? (
        <div className="modal-backdrop" role="presentation">
          <section className="auth-dialog" role="dialog" aria-modal="true" aria-labelledby="auth-title">
            <div className="section-title">
              <div>
                <span>계정</span>
                <h2 id="auth-title">{authMode === "login" ? "로그인" : "회원가입"}</h2>
              </div>
              <button className="mini-command" onClick={() => setAuthDialogOpen(false)}>
                닫기
              </button>
            </div>
            <div className="auth-panel">
              <input
                value={authUsername}
                onChange={(event) => setAuthUsername(event.target.value)}
                placeholder="ID"
                aria-label="회원 ID"
              />
              <input
                value={authPassword}
                onChange={(event) => setAuthPassword(event.target.value)}
                placeholder="Password"
                type="password"
                aria-label="비밀번호"
              />
              {authMode === "register" ? (
                <input
                  value={authNickname}
                  onChange={(event) => setAuthNickname(event.target.value)}
                  placeholder="닉네임"
                  aria-label="닉네임"
                />
              ) : null}
              <div className="auth-actions">
                <button
                  onClick={() =>
                    submitAuth(authMode, authUsername, authPassword, authNickname)
                      .then((session) => {
                        setAuthSession(session);
                        setAuthPassword("");
                        setAuthDialogOpen(false);
                        setRoomMessage(
                          authMode === "register"
                            ? `${session.user.nickname} 계정으로 가입하고 로그인했습니다.`
                            : `${session.user.nickname} 계정으로 로그인했습니다.`
                        );
                      })
                      .catch(() =>
                        setRoomMessage(
                          authMode === "register"
                            ? "회원가입에 실패했습니다. ID와 비밀번호를 확인해주세요."
                            : "로그인에 실패했습니다. ID와 비밀번호를 확인해주세요."
                        )
                      )
                  }
                >
                  {authMode === "register" ? "회원가입" : "로그인"}
                </button>
                <button
                  onClick={() => setAuthMode(authMode === "register" ? "login" : "register")}
                >
                  {authMode === "register" ? "로그인으로" : "회원가입으로"}
                </button>
              </div>
            </div>
          </section>
        </div>
      ) : null}
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
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<GuestSession>;
    if (!isStoredPlayerSession(parsed)) {
      localStorage.removeItem(guestStorageKey);
      return null;
    }
    return parsed;
  } catch {
    localStorage.removeItem(guestStorageKey);
    return null;
  }
}

function saveGuestSession(session: GuestSession) {
  localStorage.setItem(guestStorageKey, JSON.stringify(session));
}

function readAuthSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(authStorageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<AuthSession>;
    if (!isStoredPlayerSession(parsed) || typeof parsed.user?.username !== "string") {
      localStorage.removeItem(authStorageKey);
      return null;
    }
    return parsed as AuthSession;
  } catch {
    localStorage.removeItem(authStorageKey);
    return null;
  }
}

function saveAuthSession(session: AuthSession) {
  localStorage.setItem(authStorageKey, JSON.stringify(session));
}

function isStoredPlayerSession(value: Partial<GuestSession> | Partial<AuthSession>): value is GuestSession {
  return (
    typeof value.sessionToken === "string" &&
    value.sessionToken.length > 0 &&
    typeof value.user?.id === "string" &&
    value.user.id.length > 0 &&
    typeof value.user?.nickname === "string" &&
    value.user.nickname.length > 0
  );
}

function readPlayerSession(): GuestSession | null {
  const auth = readAuthSession();
  if (auth) {
    return {
      sessionToken: auth.sessionToken,
      user: { id: auth.user.id, nickname: auth.user.nickname }
    };
  }
  return readGuestSession();
}

function normalizeRoom(room: Room): Room {
  return {
    ...room,
    participants: room.participants ?? [],
    spectators: room.spectators ?? [],
    playingPlayerIds: room.playingPlayerIds ?? [],
    options: {
      turnSeconds: room.options?.turnSeconds ?? 60
    }
  };
}

async function createGuest(apiURL: string): Promise<GuestSession> {
  const response = await fetch(`${apiURL}/api/guests`, { method: "POST" });
  if (!response.ok) throw new Error("failed to create guest");

  const session = (await response.json()) as GuestSession;
  saveGuestSession(session);
  return session;
}

async function register(username: string, password: string, nickname: string): Promise<AuthSession> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password, nickname })
  });
  if (!response.ok) throw new Error("register failed");
  const session = (await response.json()) as AuthSession;
  saveAuthSession(session);
  return session;
}

async function login(username: string, password: string): Promise<AuthSession> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password })
  });
  if (!response.ok) throw new Error("login failed");
  const session = (await response.json()) as AuthSession;
  saveAuthSession(session);
  return session;
}

function submitAuth(mode: "login" | "register", username: string, password: string, nickname: string) {
  if (mode === "register") return register(username, password, nickname);
  return login(username, password);
}

async function createRoom(
  gameId: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const game = games.find((item) => item.id === gameId);
    const data = await authorizedJSON<{ room: Room }>("/api/rooms", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ gameId, maxPlayers: Math.max(game?.maxPlayers ?? 4, 12) })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(`${data.room.code} 코드를 친구에게 공유하세요.`);
    markPresence(data.room.id, undefined, "ONLINE").catch(() => undefined);
  } catch {
    onMessage("방 생성에 실패했습니다.");
  }
}

async function quickMatch(
  gameId: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const game = games.find((item) => item.id === gameId);
    const data = await authorizedJSON<{ room: Room; matched: boolean }>("/api/match/quick", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ gameId, maxPlayers: game?.maxPlayers ?? 4 })
    });
    onRoom(normalizeRoom(data.room));
    markPresence(data.room.id, data.room.activeSessionId, "ONLINE").catch(() => undefined);
    onMessage(data.matched ? "대기 중인 방에 매칭되었습니다." : "퀵매치 방을 만들고 친구를 기다립니다.");
  } catch {
    onMessage("퀵매치에 실패했습니다.");
  }
}

async function joinRoom(
  code: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>("/api/rooms/join", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ code })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(`${data.room.code} 방에 입장했습니다.`);
    markPresence(data.room.id, data.room.activeSessionId, "ONLINE").catch(() => undefined);
  } catch {
    onMessage("방 코드를 확인해주세요.");
  }
}

async function spectateRoom(
  roomID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/spectate`, session.sessionToken, {
      method: "POST"
    });
    onRoom(normalizeRoom(data.room));
    onMessage("관전자로 입장했습니다.");
  } catch {
    onMessage("관전 입장에 실패했습니다.");
  }
}

async function updateRoomOptions(
  roomID: string,
  turnSeconds: number,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/options`, session.sessionToken, {
      method: "PATCH",
      body: JSON.stringify({ turnSeconds })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(`턴 제한시간을 ${data.room.options.turnSeconds}초로 변경했습니다.`);
  } catch {
    onMessage("방장만 옵션을 바꿀 수 있습니다.");
  }
}

async function kickPlayer(
  roomID: string,
  userID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) return;
  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/kick`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ userId: userID })
    });
    onRoom(normalizeRoom(data.room));
    onMessage("선택한 사용자를 방에서 내보냈습니다.");
  } catch {
    onMessage("방장만 강퇴할 수 있습니다.");
  }
}

async function transferHost(
  roomID: string,
  userID: string,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) return;
  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/transfer-host`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ userId: userID })
    });
    onRoom(normalizeRoom(data.room));
    onMessage("방장을 위임했습니다.");
  } catch {
    onMessage("방장만 권한을 위임할 수 있습니다.");
  }
}

async function cancelQuickMatch(roomID: string, gameId: string, onMessage: (message: string) => void) {
  const session = readPlayerSession();
  if (!session) return;
  try {
    const data = await authorizedJSON<{ cancelled: boolean }>("/api/match/cancel", session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ roomId: roomID, gameId })
    });
    onMessage(data.cancelled ? "퀵매치 대기를 취소했습니다." : "이미 매칭 대기열에 없습니다.");
  } catch {
    onMessage("퀵매치 취소에 실패했습니다.");
  }
}

async function sendChat(roomID: string, text: string, kind: "chat" | "emoji"): Promise<ChatMessage> {
  const guest = readPlayerSession();
  if (!guest) throw new Error("missing guest");
  const data = await authorizedJSON<{ message: ChatMessage }>(`/api/rooms/${roomID}/chat`, guest.sessionToken, {
    method: "POST",
    body: JSON.stringify({ text, kind })
  });
  return data.message;
}

async function fetchRoomChat(roomID: string): Promise<ChatMessage[]> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/rooms/${roomID}/chat`);
  if (!response.ok) throw new Error("failed to fetch chat");
  const data = (await response.json()) as { messages: ChatMessage[] };
  return data.messages;
}

async function fetchRoom(roomID: string): Promise<Room> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/rooms/${roomID}`);
  if (!response.ok) throw new Error("failed to fetch room");
  const data = (await response.json()) as { room: Room };
  return normalizeRoom(data.room);
}

async function markPresence(
  roomID: string,
  sessionID: string | undefined,
  status: "ONLINE" | "DISCONNECTED"
) {
  const guest = readPlayerSession();
  if (!guest) return;
  await authorizedJSON<{ presence: Presence }>(`/api/rooms/${roomID}/presence`, guest.sessionToken, {
    method: "POST",
    body: JSON.stringify({ sessionId: sessionID, status })
  });
}

async function fetchLeaderboard(gameId: string): Promise<LeaderboardRow[]> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/leaderboard?gameId=${gameId}&limit=5`);
  if (!response.ok) throw new Error("failed to fetch leaderboard");
  const data = (await response.json()) as { rows: LeaderboardRow[] };
  return data.rows;
}

async function fetchTutorial(gameId: string): Promise<TutorialGuide> {
  const apiURL = import.meta.env.VITE_API_URL ?? "http://localhost:4000";
  const response = await fetch(`${apiURL}/api/tutorials/${gameId}`);
  if (!response.ok) throw new Error("failed to fetch tutorial");
  const data = (await response.json()) as { guide: TutorialGuide };
  return data.guide;
}

async function setReady(
  roomID: string,
  ready: boolean,
  onRoom: (room: Room) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/ready`, session.sessionToken, {
      method: "POST",
      body: JSON.stringify({ ready })
    });
    onRoom(normalizeRoom(data.room));
    onMessage(ready ? "준비 완료했습니다." : "준비를 취소했습니다.");
  } catch {
    onMessage("Ready 변경에 실패했습니다.");
  }
}

async function startGame(
  roomID: string,
  playerIds: string[],
  onRoom: (room: Room) => void,
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void
) {
  const session = readPlayerSession();
  if (!session) {
    onMessage("게스트 세션을 준비하는 중입니다.");
    return;
  }

  try {
    const data = await authorizedJSON<{ room: Room; session: GameSession }>(
      `/api/rooms/${roomID}/start`,
      session.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({ playerIds })
      }
    );
    onRoom(normalizeRoom(data.room));
    onSession(data.session);
    onMessage("선택된 플레이어로 게임을 시작했습니다.");
  } catch {
    onMessage("선택 인원과 Ready 상태를 확인해주세요.");
  }
}

async function sendGameAction(
  sessionID: string,
  roomID: string | undefined,
  type:
    | "davinci.pass"
    | "davinci.finish"
    | "halli-galli.flip"
    | "halli-galli.ring"
    | "splendor.take_token"
    | "splendor.buy_card"
    | "dalmuti.play"
    | "dalmuti.pass"
    | "werewolf.see_player"
    | "werewolf.see_center"
    | "werewolf.rob"
    | "werewolf.troublemake"
    | "werewolf.drunk_swap"
    | "werewolf.finish_night"
    | "werewolf.vote"
    | "rummikub.meld"
    | "rummikub.draw"
    | "bang.draw"
    | "bang.play"
    | "bang.end_turn"
    | "sutda.call"
    | "sutda.fold"
    | "sutda.showdown"
    | "gostop.play"
    | "gostop.go"
    | "gostop.stop"
    | "onecard.play"
    | "onecard.draw"
    | "jokerdraw.draw",
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void,
  payload?: Record<string, unknown>
) {
  const guest = readPlayerSession();
  if (!guest || !roomID) return;

  try {
    const data = await authorizedJSON<{ session: GameSession }>(
      `/api/sessions/${sessionID}/actions`,
      guest.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({
          roomId: roomID,
          type,
          payload,
          clientRequestId: crypto.randomUUID()
        })
      }
    );
    onSession(data.session);
    onMessage("액션이 반영되었습니다.");
  } catch {
    onMessage("지금은 해당 액션을 할 수 없습니다.");
  }
}

async function sendGuessAction(
  sessionID: string,
  roomID: string | undefined,
  payload: {
    targetPlayerId: string;
    tileIndex: number;
    color: "black" | "white";
    value: number;
  },
  onSession: (session: GameSession) => void,
  onMessage: (message: string) => void
) {
  const guest = readPlayerSession();
  if (!guest || !roomID || !payload.targetPlayerId) return;

  try {
    const data = await authorizedJSON<{ session: GameSession }>(
      `/api/sessions/${sessionID}/actions`,
      guest.sessionToken,
      {
        method: "POST",
        body: JSON.stringify({
          roomId: roomID,
          type: "davinci.guess",
          payload,
          clientRequestId: crypto.randomUUID()
        })
      }
    );
    onSession(data.session);
    onMessage("추측 결과가 반영되었습니다.");
  } catch {
    onMessage("추측할 수 없는 대상이거나 내 차례가 아닙니다.");
  }
}

async function roomPostAction(
  roomID: string | undefined,
  action: "rematch" | "return-lobby" | "leave",
  onRoom: (room: Room | null) => void,
  afterAction: () => void,
  onMessage: (message: string) => void
) {
  const guest = readPlayerSession();
  if (!guest || !roomID) return;

  const data = await authorizedJSON<{ room: Room }>(`/api/rooms/${roomID}/${action}`, guest.sessionToken, {
    method: "POST"
  });

  if (action === "leave" || data.room.status === "CLOSED") {
    onRoom(null);
    onMessage("방에서 나왔습니다.");
  } else {
    onRoom(normalizeRoom(data.room));
    onMessage(action === "rematch" ? "다시 하기 준비 로비로 돌아왔습니다." : "로비로 돌아왔습니다.");
  }
  afterAction();
}

async function fetchSession(sessionID: string, token: string): Promise<GameSession> {
  const data = await authorizedJSON<{ session: GameSession }>(`/api/sessions/${sessionID}`, token, {
    method: "GET"
  });
  return data.session;
}

function participantName(room: Room | null, playerID: string) {
  return room?.participants.find((participant) => participant.user.id === playerID)?.user.nickname ?? "상대";
}

function fruitLabel(fruit: string) {
  const labels: Record<string, string> = {
    banana: "바나나",
    strawberry: "딸기",
    lime: "라임",
    plum: "자두"
  };
  return labels[fruit] ?? fruit;
}

function gemLabel(color: string) {
  const labels: Record<string, string> = {
    white: "흰색",
    blue: "파랑",
    green: "초록",
    red: "빨강",
    black: "검정"
  };
  return labels[color] ?? color;
}

function formatCost(cost: Record<string, number>) {
  const parts = Object.entries(cost)
    .filter(([, value]) => value > 0)
    .map(([color, value]) => `${gemLabel(color)} ${value}`);
  return parts.length > 0 ? parts.join(" · ") : "없음";
}

function groupDalmutiHand(hand: HandCard[]) {
  const counts = new Map<number, number>();
  hand.forEach((card) => {
    if (typeof card.rank === "number") counts.set(card.rank, (counts.get(card.rank) ?? 0) + 1);
  });
  return Array.from(counts.entries())
    .map(([rank, count]) => ({ rank, count }))
    .sort((left, right) => left.rank - right.rank);
}

function dalmutiRankLabel(rank: number) {
  if (rank === 1) return "1 달무티";
  if (rank === 13) return "광대";
  return `${rank} 계급`;
}

function werewolfRoleLabel(role: string) {
  const labels: Record<string, string> = {
    werewolf: "늑대인간",
    seer: "예언자",
    robber: "강도",
    troublemaker: "말썽쟁이",
    drunk: "주정뱅이",
    insomniac: "불면증",
    villager: "마을 주민",
    hidden: "비공개"
  };
  return labels[role] ?? role;
}

function werewolfPhaseLabel(phase: GameSession["state"]["phase"]) {
  if (phase === "NIGHT") return "밤";
  if (phase === "DISCUSSION") return "토론/투표";
  if (phase === "FINISHED") return "종료";
  return "대기";
}

function bangCardLabel(type: string) {
  const labels: Record<string, string> = {
    bang: "BANG!",
    missed: "빗맞음",
    beer: "맥주",
    gatling: "개틀링"
  };
  return labels[type] ?? type;
}

function bangRoleLabel(role: string) {
  const labels: Record<string, string> = {
    sheriff: "보안관",
    deputy: "부관",
    outlaw: "무법자",
    renegade: "배신자"
  };
  return labels[role] ?? role;
}

function bangWinnerLabel(winner: string) {
  const labels: Record<string, string> = {
    law: "보안관/부관",
    outlaw: "무법자",
    renegade: "배신자"
  };
  return labels[winner] ?? "미정";
}

function goStopKindLabel(kind: string) {
  const labels: Record<string, string> = {
    bright: "광",
    animal: "열끗",
    ribbon: "띠",
    junk: "피"
  };
  return labels[kind] ?? kind;
}

function standardCardLabel(card: HandCard) {
  if (card.joker || card.id === "joker") return "Joker";
  const rankLabels: Record<string, string> = {
    1: "A",
    11: "J",
    12: "Q",
    13: "K"
  };
  const rank = card.rank ? rankLabels[card.rank] ?? String(card.rank) : "?";
  return `${standardSuitLabel(card.suit)} ${rank}`.trim();
}

function standardSuitLabel(suit?: string) {
  const labels: Record<string, string> = {
    spade: "♠",
    heart: "♥",
    diamond: "♦",
    club: "♣"
  };
  return suit ? labels[suit] ?? suit : "";
}

function toggleSelected(values: string[], target: string) {
  if (values.includes(target)) {
    return values.filter((value) => value !== target);
  }
  return [...values, target];
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

  if (!response.ok) {
    if (response.status === 401 && readAuthSession()?.sessionToken === token) {
      localStorage.removeItem(authStorageKey);
    }
    throw new Error(`request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}
