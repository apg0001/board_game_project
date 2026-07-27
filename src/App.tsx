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
import {
  BangCardFace,
  DalmutiCardFan,
  HwatuCardFace,
  PlayingCardBack,
  PlayingCardFace,
  SplendorCardFace,
  SplendorNobleFace
} from "./components/CardFace";
import { FlowOptionCard } from "./components/FlowOptionCard";
import { fallbackRecommendedGames, games, playerCount } from "./domain/gameCatalog";
import type {
  ApiGame,
  AuthSession,
  ChatMessage,
  DavinciPlayer,
  DavinciTile,
  GameCategory,
  GameSession,
  GuestSession,
  HandCard,
  LeaderboardRow,
  LobbyFlow,
  OneCardRules,
  Presence,
  Room,
  RummikubTile,
  TutorialGuide
} from "./domain/types";
import {
  authorizedJSON,
  createGuest,
  createRoom,
  fetchLeaderboard,
  fetchPublicRooms,
  fetchRoom,
  fetchRoomChat,
  fetchSession,
  fetchTutorial,
  joinPublicRoom,
  joinRoom,
  kickPlayer,
  markPresence,
  quickMatch,
  roomPostAction,
  sendChat,
  sendGameAction,
  sendGuessAction,
  setReady,
  spectateRoom,
  startGame,
  submitAuth,
  transferHost,
  updateRoomOptions,
  voteRoomRules
} from "./lib/api";
import { head, listOf, tail } from "./lib/collections";
import { appendChatMessage, normalizeRoom } from "./lib/normalizers";
import { nicknameInitial, safeNickname } from "./lib/playerIdentity";
import { parseRealtimeMessage } from "./lib/realtime";
import {
  authStorageKey,
  readAuthSession,
  readGuestSession,
  readPlayerSession,
  roomStorageKey,
  saveAuthSession,
  saveGuestSession,
  sessionStorageKey
} from "./lib/sessionStorage";

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
  const [guessJoker, setGuessJoker] = useState(false);
  const [pendingInsertIndex, setPendingInsertIndex] = useState(0);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [presenceByUser, setPresenceByUser] = useState<Record<string, Presence>>({});
  const [leaderboard, setLeaderboard] = useState<LeaderboardRow[]>([]);
  const [publicRooms, setPublicRooms] = useState<Room[]>([]);
  const [selectedGameId, setSelectedGameId] = useState("davinci");
  const [tutorialGuide, setTutorialGuide] = useState<TutorialGuide | null>(null);
  const [gameSearch, setGameSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("전체");
  const [turnSeconds, setTurnSeconds] = useState(60);
  const [roomMaxPlayers, setRoomMaxPlayers] = useState(4);
  const [maxWaitSeconds, setMaxWaitSeconds] = useState(180);
  const [autoStart, setAutoStart] = useState(false);
  const [allowSpectators, setAllowSpectators] = useState(true);
  const [davinciAdvancedDash, setDavinciAdvancedDash] = useState("off");
  const [davinciTournamentScoring, setDavinciTournamentScoring] = useState("off");
  const [onecardAttackCards, setOnecardAttackCards] = useState("two-ace-joker");
  const [onecardDefenseMode, setOnecardDefenseMode] = useState("attack-or-joker");
  const [onecardJokerDrawCount, setOnecardJokerDrawCount] = useState("5");
  const [onecardStacking, setOnecardStacking] = useState("on");
  const [onecardChangeSuitCards, setOnecardChangeSuitCards] = useState("seven-joker");
  const [onecardPenalty, setOnecardPenalty] = useState("on");
  const [onecardFinalAttack, setOnecardFinalAttack] = useState("on");
  const [onecardFinalSpecial, setOnecardFinalSpecial] = useState("on");
  const [onecardDeclaredSuit, setOnecardDeclaredSuit] = useState("spade");
  const [roomCodeInput, setRoomCodeInput] = useState("");
  const [roomMessage, setRoomMessage] = useState("방을 만들거나 초대 코드를 입력하세요.");
  const [selectedTileIds, setSelectedTileIds] = useState<string[]>([]);
  const [pendingRummikubGroups, setPendingRummikubGroups] = useState<string[][]>([]);
  const [selectedPlayerIds, setSelectedPlayerIds] = useState<string[]>([]);
  const [selectedGemColors, setSelectedGemColors] = useState<string[]>([]);
  const [selectedBangTargetId, setSelectedBangTargetId] = useState("");
  const [selectedBangDiscardIds, setSelectedBangDiscardIds] = useState<string[]>([]);
  const [authDialogOpen, setAuthDialogOpen] = useState(false);
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const [lobbyFlow, setLobbyFlow] = useState<LobbyFlow>("home");
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
          if (
            !current ||
            (current.user.id === data.user.id &&
              current.user.nickname === data.user.nickname &&
              current.user.role === data.user.role)
          ) {
            return current;
          }
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
      const message = parseRealtimeMessage(event.data);
      if (!message) return;
      if (message.type === "room.updated") {
        const nextRoom = message.payload.room;
        if (!nextRoom) return;
        setCurrentRoom(normalizeRoom(nextRoom));
        setRoomMessage(`${nextRoom.code} 방 상태가 갱신되었습니다.`);
      }
      if (message.type === "chat.message" && message.payload.message) {
        setChatMessages((previous) => appendChatMessage(previous, message.payload.message));
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
    if (currentRoom || (lobbyFlow !== "public-room" && lobbyFlow !== "game-rooms")) return;

    const gameID = lobbyFlow === "game-rooms" ? selectedGameId : "";
    fetchPublicRooms(gameID).then(setPublicRooms).catch(() => setPublicRooms([]));
    const timer = window.setInterval(() => {
      fetchPublicRooms(gameID).then(setPublicRooms).catch(() => undefined);
    }, 5000);

    return () => window.clearInterval(timer);
  }, [currentRoom, lobbyFlow, selectedGameId]);

  useEffect(() => {
    if (currentRoom) {
      setLobbyFlow("room");
    } else if (lobbyFlow === "room") {
      setLobbyFlow("home");
    }
  }, [currentRoom, lobbyFlow]);

  useEffect(() => {
    if (!currentRoom) return;

    setTurnSeconds(currentRoom.options?.turnSeconds ?? 60);
    setRoomMaxPlayers(currentRoom.maxPlayers ?? 4);
    setMaxWaitSeconds(currentRoom.options?.maxWaitSeconds ?? 180);
    setAutoStart(currentRoom.options?.autoStart ?? false);
    setAllowSpectators(currentRoom.options?.allowSpectators ?? true);
  }, [
    currentRoom?.id,
    currentRoom?.options?.turnSeconds,
    currentRoom?.options?.maxWaitSeconds,
    currentRoom?.options?.autoStart,
    currentRoom?.options?.allowSpectators
  ]);

  useEffect(() => {
    const vote = currentRoom?.ruleVotes?.find((item) => item.userId === playerID);
    if (!vote) return;
    setOnecardAttackCards(vote.choices.attackCards ?? "two-ace-joker");
    setOnecardDefenseMode(vote.choices.defenseMode ?? "attack-or-joker");
    setOnecardJokerDrawCount(vote.choices.jokerDrawCount ?? "5");
    setOnecardStacking(vote.choices.stacking ?? "on");
  }, [currentRoom?.ruleVotes, playerID]);

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
      const valid = listOf(current).filter((id) => participantIDs.includes(id));
      if (valid.length > 0) return head(valid, game.maxPlayers);
      return head(participantIDs, game.maxPlayers);
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
    if (!currentSession || !playerID || !playerToken) return;

    const wsURL = import.meta.env.VITE_WS_URL ?? "ws://localhost:4000/ws";
    const socket = new WebSocket(
      `${wsURL}?room=${encodeURIComponent(`game:${currentSession.id}`)}&user=${encodeURIComponent(
        playerID
      )}`
    );

    socket.onmessage = (event) => {
      const message = parseRealtimeMessage(event.data);
      if (!message) return;
      if (message.type !== "game.updated") return;
      fetchSession(currentSession.id, playerToken)
        .then(setCurrentSession)
        .catch(() => {
          if (message.payload.session) {
            setCurrentSession(message.payload.session);
          }
        });
    };

    return () => {
      socket.close();
    };
  }, [currentSession?.id, playerID, playerToken]);

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
        mark: fallback?.mark ?? safeNickname(apiGame.title, "?").slice(0, 1)
      };
    });

    return source
      .filter((game) => game.title.includes(gameSearch.trim()) || gameSearch.trim() === "")
      .filter((game) => categoryFilter === "전체" || game.categories.includes(categoryFilter as GameCategory));
  }, [apiGames, categoryFilter, gameSearch]);

  const activePlayerCount = currentRoom?.participants.length ?? 0;
  const recommendationPlayerCount = currentRoom ? activePlayerCount : playerCount;
  const partyMembers = currentRoom?.participants ?? [];
  const me = currentRoom?.participants.find(
    (participant) => participant.user.id === playerID
  );
  const davinciPlayers = currentSession?.state.players ?? [];
  const myDavinciPlayer = davinciPlayers.find((player) => player.playerId === playerID);
  const opponentPlayers = davinciPlayers.filter((player) => player.playerId !== playerID);
  const targetPlayer = opponentPlayers.find((player) => player.playerId === guessTarget) ?? opponentPlayers[0];
  const currentTurnPlayer = davinciPlayers[currentSession?.state.currentPlayerIndex ?? 0];
  const isSessionFinished = currentSession?.status === "FINISHED" || Boolean(currentSession?.state.finished);
  const isMyTurn = Boolean(!isSessionFinished && currentTurnPlayer?.playerId === playerID);
  useEffect(() => {
    if (!isMyTurn) {
      setSelectedGemColors([]);
    }
  }, [isMyTurn]);
  const selectedGame = games.find((game) => game.id === selectedGameId) ?? games[4];
  const selectedRoomGame = games.find((game) => game.id === currentRoom?.gameId) ?? selectedGame;
  const setupGame = currentRoom ? selectedRoomGame : selectedGame;
  const selectedPlayerSet = new Set(selectedPlayerIds);
  const selectedPlayers = partyMembers.filter((participant) => selectedPlayerSet.has(participant.user.id));
  const selectedReady = selectedPlayers.length > 0 && selectedPlayers.every((participant) => participant.ready);
  const selectedCountValid =
    selectedPlayers.length >= selectedRoomGame.minPlayers && selectedPlayers.length <= selectedRoomGame.maxPlayers;
  const readyCount = partyMembers.filter((participant) => participant.ready).length;
  const ruleVoteCount = currentRoom?.ruleVotes?.length ?? 0;
  const amHost = Boolean(me?.host);
  const canStartGame = Boolean(currentRoom && amHost && selectedReady && selectedCountValid && currentRoom.status === "LOBBY");
  const showHomeFlow = !currentRoom && lobbyFlow === "home";
  const flowTitle =
    lobbyFlow === "private-room"
      ? "프라이빗 방 만들기"
      : lobbyFlow === "public-room"
        ? "공개 방 만들기"
      : lobbyFlow === "quick-match"
        ? "랜덤 방 입장"
        : lobbyFlow === "game-rooms"
          ? "게임별 방 입장"
          : "방 로비";
  const flowDescription =
    lobbyFlow === "private-room"
      ? "게임을 고른 뒤 비공개 입장 코드를 친구에게 공유하세요."
      : lobbyFlow === "public-room"
        ? "누구나 목록에서 보고 들어올 수 있는 공개 방을 만들거나 입장 코드를 공유하세요."
      : lobbyFlow === "quick-match"
        ? "원하는 게임을 고르면 같은 게임을 기다리는 플레이어와 바로 매칭됩니다."
        : lobbyFlow === "game-rooms"
          ? "특정 게임만 필터링해서 해당 게임을 기다리는 방으로 입장합니다."
          : "참가자 준비, 관전, 채팅, 게임 시작을 이 화면에서 관리합니다.";
  const currentTurnName = currentTurnPlayer ? participantName(currentRoom, currentTurnPlayer.playerId) : "대기 중";
  const turnBadgeLabel = isSessionFinished ? "게임 종료" : isMyTurn ? "내 차례" : "상대 차례";
  const splendorColors = ["white", "blue", "green", "red", "black"];
  const splendorMe = davinciPlayers.find((player) => player.playerId === playerID);
  const splendorReturnCount = currentSession?.state.pendingReturnCount ?? 0;
  const splendorMustReturnTokens = Boolean(currentSession?.state.pendingReturnPlayerId === playerID && splendorReturnCount > 0);
  const splendorPendingNobleChoices = currentSession?.state.pendingNobleChoices ?? [];
  const splendorMustChooseNoble = Boolean(
    currentSession?.state.pendingNoblePlayerId === playerID && splendorPendingNobleChoices.length > 0
  );
  const splendorActionBlocked = splendorReturnCount > 0 || splendorPendingNobleChoices.length > 0;
  const splendorTokenColors = splendorMustReturnTokens ? [...splendorColors, "gold"] : splendorColors;
  const splendorTierRows = [3, 2, 1]
    .map((tier) => ({
      tier,
      cards: currentSession?.state.markets?.[String(tier)] ?? [],
      deckSize: currentSession?.state.decks?.[String(tier)]?.length ?? 0
    }))
    .filter((row) => row.cards.length > 0 || row.deckSize > 0);
  const splendorMarketRows =
    splendorTierRows.length > 0
      ? splendorTierRows
      : [{ tier: 1, cards: currentSession?.state.market ?? [], deckSize: currentSession?.state.deck?.length ?? 0 }];
  const dalmutiMe = davinciPlayers.find((player) => player.playerId === playerID);
  const dalmutiGroups = groupDalmutiHand(dalmutiMe?.hand ?? []);
  const werewolfMe = davinciPlayers.find((player) => player.playerId === playerID);
  const werewolfOthers = davinciPlayers.filter((player) => player.playerId !== playerID);
  const rummikubMe = davinciPlayers.find((player) => player.playerId === playerID);
  const rummikubPendingTileIds = new Set(pendingRummikubGroups.flat());
  const rummikubSubmitGroups = [...pendingRummikubGroups, ...(selectedTileIds.length >= 3 ? [selectedTileIds] : [])];
  const bangMe = davinciPlayers.find((player) => player.playerId === playerID);
  const bangTargets = davinciPlayers.filter((player) => player.playerId !== playerID && player.alive !== false);
  const bangSelectedTargetId = bangTargets.some((player) => player.playerId === selectedBangTargetId)
    ? selectedBangTargetId
    : (bangTargets[0]?.playerId ?? "");
  const bangAliveCount = davinciPlayers.filter((player) => player.alive !== false).length;
  const bangPendingAttack = currentSession?.gameId === "bang" ? currentSession.state.pendingAttack : undefined;
  const bangMustRespond = Boolean(bangPendingAttack?.targetPlayerId === playerID);
  const bangHasMissed = Boolean((bangMe?.hand ?? []).some((card) => card.type === "missed"));
  const bangPendingDiscardPlayerId = currentSession?.gameId === "bang" ? currentSession.state.pendingDiscardPlayerId : undefined;
  const bangPendingDiscardCount = currentSession?.gameId === "bang" ? (currentSession.state.pendingDiscardCount ?? 0) : 0;
  const bangMustDiscard = Boolean(bangPendingDiscardPlayerId === playerID && bangPendingDiscardCount > 0);
  const bangDiscardSelectedIds = selectedBangDiscardIds.filter((id) => (bangMe?.hand ?? []).some((card) => card.id === id));
  const bangActionBlocked = Boolean(bangPendingAttack || bangPendingDiscardPlayerId);
  const sutdaMe = davinciPlayers.find((player) => player.playerId === playerID);
  const gostopMe = davinciPlayers.find((player) => player.playerId === playerID);
  const onecardMe = davinciPlayers.find((player) => player.playerId === playerID);
  const onecardTopCard = currentSession?.state.discardPile?.[(currentSession.state.discardPile?.length ?? 0) - 1];
  const jokerdrawMe = davinciPlayers.find((player) => player.playerId === playerID);
  const jokerdrawNextTarget =
    currentSession?.gameId === "jokerdraw"
      ? nextActivePlayerAfter(davinciPlayers, currentSession.state.currentPlayerIndex ?? 0)
      : undefined;
  const jokerdrawTargets =
    jokerdrawNextTarget && jokerdrawNextTarget.playerId !== currentTurnPlayer?.playerId ? [jokerdrawNextTarget] : [];

  return (
    <main className="app-shell">
      <section className={`hero-panel ${currentSession ? "compact-hero" : ""}`} aria-label="게임 로비">
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
            <button
              className="profile-button"
              aria-label="내 프로필"
              onClick={() => {
                if (!authSession) {
                  setAuthMode("login");
                  setAuthDialogOpen(true);
                }
              }}
            >
              {nicknameInitial(playerNickname, "G")}
            </button>
          </div>
        </nav>

        <div className="hero-grid">
          <div className="hero-copy">
            <p className="eyebrow">
              <Wifi size={16} aria-hidden="true" />
              모바일, 태블릿, 데스크톱 동시 플레이 · 서버 {serverStatus}
            </p>
            <h1>어떻게 모일지 먼저 고르고, 다음 화면에서 게임을 시작하세요.</h1>
            <p className="summary">
              초대 방, 랜덤 매칭, 게임별 입장을 분리해 모바일에서도 헷갈리지 않는 흐름으로 플레이합니다.
            </p>
            <div className="hero-actions">
              <button className="primary-button" onClick={() => setLobbyFlow("private-room")}>
                <Plus size={18} />
                프라이빗
              </button>
              <button className="secondary-button" onClick={() => setLobbyFlow("public-room")}>
                <UsersRound size={18} />
                공개 방
              </button>
              <button className="secondary-button" onClick={() => setLobbyFlow("game-rooms")}>
                <Search size={18} />
                게임별 방
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

      {showHomeFlow ? (
        <section className="flow-home" aria-label="플레이 방식 선택">
          <FlowOptionCard
            primary
            icon={<Plus size={22} />}
            title="프라이빗 방 만들기"
            description="친구들과 코드로 들어오는 비공개 방을 만듭니다."
            onClick={() => setLobbyFlow("private-room")}
          />
          <FlowOptionCard
            icon={<UsersRound size={22} />}
            title="공개 방 만들기"
            description="목록에 노출되는 방을 만들거나 공개 방에 직접 들어갑니다."
            onClick={() => setLobbyFlow("public-room")}
          />
          <FlowOptionCard
            icon={<Search size={22} />}
            title="특정 게임 방만 들어가기"
            description="게임 목록에서 하나를 고른 뒤 해당 게임 매칭으로 입장합니다."
            onClick={() => setLobbyFlow("game-rooms")}
          />
          <FlowOptionCard
            icon={<Play size={22} />}
            title="퀵매칭으로 랜덤 방"
            description="방을 고르지 않고 선택한 게임의 대기열로 바로 들어갑니다."
            onClick={() => setLobbyFlow("quick-match")}
          />
        </section>
      ) : null}

      {!showHomeFlow ? (
      <section className={`content-grid ${currentSession ? "playing-content-grid" : ""}`}>
        <div className="flow-page-header">
          <div>
            <span className="eyebrow compact">
              <Sparkles size={15} />
              {currentRoom ? "현재 방" : "다음 단계"}
            </span>
            <h2>{flowTitle}</h2>
            <p>{flowDescription}</p>
          </div>
          {!currentRoom ? (
            <button
              className="mini-command"
              onClick={() => {
                setLobbyFlow("home");
                setRoomMessage("방을 만들거나 입장 방식을 선택하세요.");
              }}
            >
              홈으로
            </button>
          ) : null}
        </div>
        <aside className="control-panel" aria-label="매칭 패널">
          <div className="room-card">
            <div className="section-title">
              <div>
                <span>{playerNickname || "게스트 준비 중"}</span>
                <h2>
                  {currentRoom
                    ? `${activePlayerCount}명 입장 중 · Ready ${readyCount}/${partyMembers.length}`
                    : "아직 방 없음"}
                </h2>
              </div>
              <UsersRound size={22} />
            </div>
            <div className="party-list">
              {currentRoom ? (
                partyMembers.map((participant) => (
                  <div className="party-member" key={participant.user.id}>
                    <span>{nicknameInitial(participant.user.nickname, "U")}</span>
                    <div>
                      <strong>{safeNickname(participant.user.nickname, "플레이어")}</strong>
                      <small>
                        {participant.host ? "방장" : participant.ready ? "준비 완료" : "대기 중"}
                        {" · "}
                        {presenceByUser[participant.user.id]?.status === "DISCONNECTED" ? "재접속 대기" : "온라인"}
                        {currentRoom.status === "PLAYING" && !currentRoom.playingPlayerIds?.includes(participant.user.id)
                          ? " · 이번 판 관전"
                          : ""}
                      </small>
                      {amHost && participant.user.id !== playerID ? (
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
                ))
              ) : (
                <div className="empty-party-state">
                  <strong>방을 만들거나 코드로 입장하세요.</strong>
                  <small>참가자 수와 Ready 상태는 실제 방에 들어간 뒤 표시됩니다.</small>
                </div>
              )}
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
                <span>{currentRoom?.visibility === "PUBLIC" ? "공개 방 입장 코드" : "프라이빗 방 입장 코드"}</span>
                <h2>{currentRoom?.code ?? "없음"}</h2>
                <small className="status-badge">
                  {currentRoom ? `${currentRoom.visibility} · ${currentRoom.status}` : "HOME"}
                </small>
              </div>
              <LockKeyhole size={22} />
            </div>
            <div className="room-actions">
              <div className="selected-game-strip">
                <span>선택 게임</span>
                <strong>{setupGame.title}</strong>
                <small>
                  게임 {setupGame.players} · 방 최대 {currentRoom?.maxPlayers ?? roomMaxPlayers}명
                </small>
              </div>
              <p className="room-helper">
                게임 최대 인원을 넘으면 방장이 이번 판 플레이어를 고르고 나머지는 같은 방에서 관전합니다.
              </p>
              {!currentRoom && (lobbyFlow === "private-room" || lobbyFlow === "public-room") ? (
                <>
                  <label className="option-grid solo-option">
                    <span>
                      방 최대 인원
                      <input
                        type="number"
                        min={selectedGame.minPlayers}
                        max="12"
                        value={roomMaxPlayers}
                        onChange={(event) => setRoomMaxPlayers(Number(event.target.value))}
                      />
                    </span>
                  </label>
                  <button
                    className="wide-button"
                    onClick={() =>
                      createRoom(
                        selectedGameId,
                        lobbyFlow === "public-room" ? "PUBLIC" : "PRIVATE",
                        roomMaxPlayers,
                        setCurrentRoom,
                        setRoomMessage
                      )
                    }
                  >
                    {selectedGame.title} {lobbyFlow === "public-room" ? "공개 방 만들기" : "프라이빗 방 만들기"}
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
                </>
              ) : null}
              {!currentRoom && lobbyFlow === "quick-match" ? (
                <button className="wide-button play-now" onClick={() => quickMatch(selectedGameId, setCurrentRoom, setRoomMessage)}>
                  {selectedGame.title} 랜덤 방 입장
                  <ChevronRight size={18} />
                </button>
              ) : null}
              {!currentRoom && lobbyFlow === "game-rooms" ? (
                <button className="wide-button play-now" onClick={() => quickMatch(selectedGameId, setCurrentRoom, setRoomMessage)}>
                  {selectedGame.title} 방만 찾기
                  <ChevronRight size={18} />
                </button>
              ) : null}
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
                <div className="room-options-panel">
                  <div className="option-grid">
                    <label>
                      방 최대 인원
                      <input
                        type="number"
                        min={partyMembers.length || selectedRoomGame.minPlayers}
                        max="12"
                        value={roomMaxPlayers}
                        disabled={!amHost}
                        onChange={(event) => setRoomMaxPlayers(Number(event.target.value))}
                      />
                    </label>
                    <label>
                      턴 제한
                      <input
                        type="number"
                        min="10"
                        max="300"
                        value={turnSeconds}
                        disabled={!amHost}
                        onChange={(event) => setTurnSeconds(Number(event.target.value))}
                      />
                    </label>
                    <label>
                      최대 대기
                      <input
                        type="number"
                        min="30"
                        max="1800"
                        step="30"
                        value={maxWaitSeconds}
                        disabled={!amHost}
                        onChange={(event) => setMaxWaitSeconds(Number(event.target.value))}
                      />
                    </label>
                  </div>
                  <label className="toggle-row">
                    <input
                      type="checkbox"
                      checked={autoStart}
                      disabled={!amHost}
                      onChange={(event) => setAutoStart(event.target.checked)}
                    />
                    <span>전원 Ready 시 자동 시작</span>
                  </label>
                  <label className="toggle-row">
                    <input
                      type="checkbox"
                      checked={allowSpectators}
                      disabled={!amHost}
                      onChange={(event) => setAllowSpectators(event.target.checked)}
                    />
                    <span>관전 허용</span>
                  </label>
                  <button
                    className="wide-button"
                    onClick={() =>
                      updateRoomOptions(
                        currentRoom.id,
                        {
                          turnSeconds,
                          maxPlayers: roomMaxPlayers,
                          maxWaitSeconds,
                          autoStart,
                          allowSpectators
                        },
                        setCurrentRoom,
                        setRoomMessage
                      )
                    }
                    disabled={!amHost}
                  >
                    방 옵션 적용
                    <ChevronRight size={18} />
                  </button>
                </div>
              ) : null}
              {currentRoom?.gameId === "davinci" && currentRoom.status === "LOBBY" ? (
                <div className="rule-vote-panel">
                  <div className="section-title compact-title">
                    <div>
                      <span>다빈치 코드 룰 투표</span>
                      <h2>{ruleVoteCount}/{partyMembers.length}명 투표</h2>
                    </div>
                    <Dice5 size={20} />
                  </div>
                  <div className="option-grid">
                    <label>
                      대시 타일
                      <select value={davinciAdvancedDash} onChange={(event) => setDavinciAdvancedDash(event.target.value)}>
                        <option value="off">기본 24타일</option>
                        <option value="on">고급 대시 포함</option>
                      </select>
                    </label>
                    <label>
                      점수 방식
                      <select value={davinciTournamentScoring} onChange={(event) => setDavinciTournamentScoring(event.target.value)}>
                        <option value="off">단판 승패</option>
                        <option value="on">토너먼트 점수</option>
                      </select>
                    </label>
                  </div>
                  <button
                    className="wide-button"
                    onClick={() =>
                      voteRoomRules(
                        currentRoom.id,
                        {
                          advancedDashTiles: davinciAdvancedDash,
                          tournamentScoring: davinciTournamentScoring
                        },
                        setCurrentRoom,
                        setRoomMessage
                      )
                    }
                  >
                    룰 투표
                    <ChevronRight size={18} />
                  </button>
                  {currentRoom.ruleMessages?.length ? (
                    <div className="rule-message-list">
                      {currentRoom.ruleMessages.map((message) => (
                        <span key={message}>{message}</span>
                      ))}
                    </div>
                  ) : (
                    <p className="room-helper">투표 변경 시 Ready가 해제됩니다. 동률은 시작 시 랜덤으로 정해집니다.</p>
                  )}
                </div>
              ) : null}
              {currentRoom?.gameId === "onecard" && currentRoom.status === "LOBBY" ? (
                <div className="rule-vote-panel">
                  <div className="section-title compact-title">
                    <div>
                      <span>원카드 룰 투표</span>
                      <h2>{ruleVoteCount}/{partyMembers.length}명 투표</h2>
                    </div>
                    <Dice5 size={20} />
                  </div>
                  <div className="option-grid">
                    <label>
                      공격카드
                      <select value={onecardAttackCards} onChange={(event) => setOnecardAttackCards(event.target.value)}>
                        <option value="two">2만</option>
                        <option value="two-ace">2/A</option>
                        <option value="two-ace-joker">2/A/조커</option>
                      </select>
                    </label>
                    <label>
                      방어카드
                      <select value={onecardDefenseMode} onChange={(event) => setOnecardDefenseMode(event.target.value)}>
                        <option value="same-rank">같은 공격카드만</option>
                        <option value="any-attack">공격카드</option>
                        <option value="attack-or-joker">공격카드/조커</option>
                      </select>
                    </label>
                    <label>
                      조커 공격
                      <select value={onecardJokerDrawCount} onChange={(event) => setOnecardJokerDrawCount(event.target.value)}>
                        <option value="5">5장</option>
                        <option value="7">7장</option>
                        <option value="10">10장</option>
                      </select>
                    </label>
                    <label>
                      공격 누적
                      <select value={onecardStacking} onChange={(event) => setOnecardStacking(event.target.value)}>
                        <option value="on">허용</option>
                        <option value="off">없음</option>
                      </select>
                    </label>
                    <label>
                      문양 변경
                      <select value={onecardChangeSuitCards} onChange={(event) => setOnecardChangeSuitCards(event.target.value)}>
                        <option value="seven-joker">7/조커</option>
                        <option value="seven">7만</option>
                        <option value="joker">조커만</option>
                        <option value="queen-joker">Q/조커</option>
                        <option value="off">없음</option>
                      </select>
                    </label>
                    <label>
                      원카드 벌칙
                      <select value={onecardPenalty} onChange={(event) => setOnecardPenalty(event.target.value)}>
                        <option value="on">사용</option>
                        <option value="off">사용 안 함</option>
                      </select>
                    </label>
                    <label>
                      막카 공격
                      <select value={onecardFinalAttack} onChange={(event) => setOnecardFinalAttack(event.target.value)}>
                        <option value="on">허용</option>
                        <option value="off">금지</option>
                      </select>
                    </label>
                    <label>
                      막카 특수
                      <select value={onecardFinalSpecial} onChange={(event) => setOnecardFinalSpecial(event.target.value)}>
                        <option value="on">허용</option>
                        <option value="off">금지</option>
                      </select>
                    </label>
                  </div>
                  <button
                    className="wide-button"
                    onClick={() =>
                      voteRoomRules(
                        currentRoom.id,
                        {
                          attackCards: onecardAttackCards,
                          defenseMode: onecardDefenseMode,
                          jokerDrawCount: onecardJokerDrawCount,
                          stacking: onecardStacking,
                          changeSuitCards: onecardChangeSuitCards,
                          oneCardPenalty: onecardPenalty,
                          allowFinalAttack: onecardFinalAttack,
                          allowFinalSpecial: onecardFinalSpecial
                        },
                        setCurrentRoom,
                        setRoomMessage
                      )
                    }
                  >
                    룰 투표
                    <ChevronRight size={18} />
                  </button>
                  {currentRoom.ruleMessages?.length ? (
                    <div className="rule-message-list">
                      {currentRoom.ruleMessages.map((message) => (
                        <span key={message}>{message}</span>
                      ))}
                    </div>
                  ) : (
                    <p className="room-helper">투표 변경 시 Ready가 해제됩니다. 동률은 시작 시 랜덤으로 정해집니다.</p>
                  )}
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
              <div className="room-status-grid" aria-label="방 상태 요약">
                <span>
                  <strong>로비</strong>
                  {currentRoom.participants.length}/{currentRoom.maxPlayers}
                </span>
                <span>
                  <strong>이번 판</strong>
                  {selectedPlayers.length}/{selectedRoomGame.maxPlayers}
                </span>
                <span>
                  <strong>관전</strong>
                  {currentRoom.spectators?.length ?? 0}
                </span>
                <span>
                  <strong>턴</strong>
                  {currentRoom.options?.turnSeconds ?? 60}초
                </span>
                <span>
                  <strong>대기</strong>
                  {currentRoom.options?.maxWaitSeconds ?? 180}초
                </span>
                <span>
                  <strong>시작</strong>
                  {currentRoom.options?.autoStart ? "자동" : "수동"}
                </span>
              </div>
            ) : null}
            {currentRoom ? (
              <p className="room-message">
                시작 조건: {selectedRoomGame.minPlayers}-{selectedRoomGame.maxPlayers}명 선택 · 선택 플레이어 Ready · 방장 시작
              </p>
            ) : null}
          </div>

          {currentSession ? (
            <div className="room-card game-session-card">
              <div className="section-title">
                <div>
                  <span>게임 세션</span>
                  <h2>{currentSession.status === "FINISHED" ? "결과 확인" : "진행 중"}</h2>
                </div>
                <Gamepad2 size={22} />
              </div>
              <div className="game-session-panel">
                <div className={`turn-banner ${isMyTurn ? "my-turn" : "waiting-turn"}`} aria-live="polite">
                  <span>{turnBadgeLabel}</span>
                  <strong>{currentTurnName}</strong>
                  <small>
                    {selectedRoomGame.title} · 라운드 {currentSession.state.round ?? 1}
                  </small>
                </div>
                <div className="session-log">
                  {tail(currentSession.state.log, 3).map((item) => (
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
                          setLobbyFlow("home");
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
                                {tail(player.faceUp, 3).map((card, index) => (
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
                          {splendorTokenColors.map((color) => {
                            const selectedCount = selectedGemColors.filter((selected) => selected === color).length;
                            const availableCount = splendorMustReturnTokens
                              ? (splendorMe?.tokens?.[color] ?? 0)
                              : (currentSession.state.bank?.[color] ?? 0);
                            return (
                              <button
                                className={`gem-button ${color} ${selectedCount > 0 ? "selected" : ""}`}
                                key={color}
                                onClick={() =>
                                  setSelectedGemColors((previous) =>
                                    splendorMustReturnTokens
                                      ? toggleReturnGemColor(previous, color, splendorReturnCount, availableCount)
                                      : toggleGemColor(previous, color)
                                  )
                                }
                                disabled={
                                  splendorMustReturnTokens
                                    ? (availableCount <= 0 && selectedCount === 0) ||
                                      (selectedGemColors.length >= splendorReturnCount && selectedCount === 0)
                                    : !isMyTurn || splendorActionBlocked || availableCount <= 0
                                }
                              >
                                <span>{gemLabel(color)}</span>
                                <strong>
                                  {availableCount}
                                  {selectedCount > 0 ? ` · 선택 ${selectedCount}` : ""}
                                </strong>
                              </button>
                            );
                          })}
                        </div>
                        <div className="room-actions">
                          <button
                            className="wide-button"
                            onClick={() =>
                              sendGameAction(
                                currentSession.id,
                                currentRoom?.id,
                                splendorMustReturnTokens ? "splendor.return_tokens" : "splendor.take_token",
                                (session) => {
                                  setCurrentSession(session);
                                  setSelectedGemColors([]);
                                },
                                setRoomMessage,
                                splendorMustReturnTokens
                                  ? { tokens: countSelectedColors(selectedGemColors) }
                                  : { colors: selectedGemColors }
                              )
                            }
                            disabled={
                              splendorMustReturnTokens
                                ? !isValidReturnGemSelection(selectedGemColors, splendorReturnCount)
                                : !isMyTurn || splendorActionBlocked || !isValidGemSelection(selectedGemColors)
                            }
                          >
                            {splendorMustReturnTokens
                              ? `초과 토큰 ${splendorReturnCount}개 반납`
                              : "보석 가져가기 (다른 색 3개 또는 같은 색 2개)"}
                            <ChevronRight size={18} />
                          </button>
                          <button
                            className="wide-button dark"
                            onClick={() => setSelectedGemColors([])}
                            disabled={selectedGemColors.length === 0}
                          >
                            선택 취소
                          </button>
                        </div>
                        <div className="splendor-market-board" aria-label="시장 카드">
                          {splendorMarketRows.map((row) => (
                            <div className="splendor-tier-row" key={`splendor-tier-${row.tier}`}>
                              <div className="splendor-tier-header">
                                <strong>{romanTier(row.tier)} 단계</strong>
                                <span>덱 {row.deckSize}장</span>
                                <button
                                  className="splendor-deck-reserve"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "splendor.reserve_card",
                                      setCurrentSession,
                                      setRoomMessage,
                                      { marketTier: row.tier, fromDeck: true }
                                    )
                                  }
                                  disabled={
                                    !isMyTurn ||
                                    splendorActionBlocked ||
                                    row.deckSize <= 0 ||
                                    (splendorMe?.reserved?.length ?? 0) >= 3
                                  }
                                >
                                  덱 예약
                                </button>
                              </div>
                              <div className="splendor-market">
                                {row.cards.map((card, index) => (
                                  <div className={`splendor-card ${card.color}`} key={card.id}>
                                    <SplendorCardFace card={card} badge={romanTier(card.tier ?? row.tier)} />
                                    <button
                                      className="mini-action-button"
                                      onClick={() =>
                                        sendGameAction(
                                          currentSession.id,
                                          currentRoom?.id,
                                          "splendor.buy_card",
                                          setCurrentSession,
                                          setRoomMessage,
                                          { marketTier: row.tier, marketIndex: index }
                                        )
                                      }
                                      disabled={!isMyTurn || splendorActionBlocked}
                                    >
                                      구매
                                    </button>
                                    <button
                                      className="mini-action-button"
                                      onClick={() =>
                                        sendGameAction(
                                          currentSession.id,
                                          currentRoom?.id,
                                          "splendor.reserve_card",
                                          setCurrentSession,
                                          setRoomMessage,
                                          { marketTier: row.tier, marketIndex: index }
                                        )
                                      }
                                      disabled={!isMyTurn || splendorActionBlocked || (splendorMe?.reserved?.length ?? 0) >= 3}
                                    >
                                      예약
                                    </button>
                                  </div>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                        {splendorMe?.reserved?.length ? (
                          <div className="splendor-market" aria-label="내 예약 카드">
                            {splendorMe.reserved.map((card, index) => (
                              <div className={`splendor-card ${card.color}`} key={`reserved-${card.id}`}>
                                <SplendorCardFace card={card} badge="예약" />
                                <button
                                  className="mini-action-button"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "splendor.buy_card",
                                      setCurrentSession,
                                      setRoomMessage,
                                      { reservedIndex: index }
                                    )
                                  }
                                  disabled={!isMyTurn || splendorActionBlocked}
                                >
                                  구매
                                </button>
                              </div>
                            ))}
                          </div>
                        ) : null}
                        <div className="splendor-nobles" aria-label="공개 귀족 타일">
                          {(currentSession.state.nobles ?? []).map((noble) => (
                            <div className="splendor-noble" key={noble.id}>
                              <SplendorNobleFace noble={noble} />
                            </div>
                          ))}
                        </div>
                        {splendorPendingNobleChoices.length > 0 ? (
                          <div className="splendor-choice-panel" aria-label="귀족 선택">
                            <strong>
                              {splendorMustChooseNoble ? "방문할 귀족 선택" : "귀족 선택 대기 중"}
                            </strong>
                            <div className="splendor-nobles">
                              {splendorPendingNobleChoices.map((noble) => (
                                <button
                                  className="splendor-noble-choice"
                                  key={`choice-${noble.id}`}
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "splendor.choose_noble",
                                      setCurrentSession,
                                      setRoomMessage,
                                      { nobleId: noble.id }
                                    )
                                  }
                                  disabled={!splendorMustChooseNoble}
                                >
                                  <SplendorNobleFace noble={noble} />
                                </button>
                              ))}
                            </div>
                          </div>
                        ) : null}
                        <div className="splendor-players">
                          {davinciPlayers.map((player) => (
                            <div className="halli-player" key={player.playerId}>
                              <strong>{participantName(currentRoom, player.playerId)}</strong>
                              <span>
                                {player.score ?? 0}점 · 카드 {player.cards?.length ?? 0}장 · 예약 {player.reserved?.length ?? 0}장 · 귀족{" "}
                                {player.nobles?.length ?? 0}명
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
                              <DalmutiCardFan rank={group.rank} count={group.count} />
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
                              disabled={rummikubPendingTileIds.has(tile.id)}
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
                        {pendingRummikubGroups.length > 0 ? (
                          <div className="rummikub-pending-groups" aria-label="등록 대기 조합">
                            {pendingRummikubGroups.map((group, groupIndex) => (
                              <div className="rummikub-pending-group" key={`pending-${groupIndex}`}>
                                <strong>묶음 {groupIndex + 1}</strong>
                                <span>{group.map((id) => rummikubTileLabel(rummikubMe?.rack ?? [], id)).join(" · ")}</span>
                                <button
                                  className="mini-action-button"
                                  onClick={() =>
                                    setPendingRummikubGroups((previous) =>
                                      previous.filter((_, index) => index !== groupIndex)
                                    )
                                  }
                                >
                                  제거
                                </button>
                              </div>
                            ))}
                          </div>
                        ) : null}
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
                          onClick={() => {
                            setPendingRummikubGroups((previous) => [...previous, selectedTileIds]);
                            setSelectedTileIds([]);
                          }}
                          disabled={!isMyTurn || selectedTileIds.length < 3}
                        >
                          선택 묶음 추가
                          <ChevronRight size={18} />
                        </button>
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
                                setPendingRummikubGroups([]);
                              },
                              setRoomMessage,
                              rummikubSubmitPayload(rummikubSubmitGroups)
                            )
                          }
                          disabled={!isMyTurn || rummikubSubmitGroups.length === 0}
                        >
                          {rummikubSubmitGroups.length > 1 ? `${rummikubSubmitGroups.length}개 조합 등록` : "선택 조합 등록"}
                          <ChevronRight size={18} />
                        </button>
                        <button
                          className="wide-button dark"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "rummikub.draw",
                              (session) => {
                                setCurrentSession(session);
                                setSelectedTileIds([]);
                                setPendingRummikubGroups([]);
                              },
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
                        {bangPendingAttack ? (
                          <div className="bang-reaction-panel" aria-label="공격 반응">
                            <strong>{bangCardLabel(bangPendingAttack.cardType)} 반응</strong>
                            <span>
                              {participantName(currentRoom, bangPendingAttack.sourcePlayerId)} →{" "}
                              {participantName(currentRoom, bangPendingAttack.targetPlayerId)}
                            </span>
                            {bangMustRespond ? (
                              <div className="bang-reaction-actions">
                                <button
                                  className="mini-action-button"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "bang.use_missed",
                                      setCurrentSession,
                                      setRoomMessage
                                    )
                                  }
                                  disabled={!bangHasMissed}
                                >
                                  빗맞음 사용
                                </button>
                                <button
                                  className="mini-action-button dark-mini-action"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "bang.take_hit",
                                      setCurrentSession,
                                      setRoomMessage
                                    )
                                  }
                                >
                                  피해 받기
                                </button>
                              </div>
                            ) : (
                              <small>{participantName(currentRoom, bangPendingAttack.targetPlayerId)} 님 선택 대기</small>
                            )}
                          </div>
                        ) : null}
                        {bangPendingDiscardPlayerId ? (
                          <div className="bang-reaction-panel" aria-label="손패 제한">
                            <strong>손패 제한</strong>
                            <span>
                              {participantName(currentRoom, bangPendingDiscardPlayerId)} 님이 카드 {bangPendingDiscardCount}장을 버려야 합니다.
                            </span>
                            {bangMustDiscard ? (
                              <button
                                className="mini-action-button dark-mini-action"
                                onClick={() =>
                                  sendGameAction(
                                    currentSession.id,
                                    currentRoom?.id,
                                    "bang.discard",
                                    (session) => {
                                      setCurrentSession(session);
                                      setSelectedBangDiscardIds([]);
                                    },
                                    setRoomMessage,
                                    { cardIds: bangDiscardSelectedIds }
                                  )
                                }
                                disabled={bangDiscardSelectedIds.length !== bangPendingDiscardCount}
                              >
                                선택 카드 버리기
                              </button>
                            ) : null}
                          </div>
                        ) : null}
                        <div className="bang-hand" aria-label="내 카드">
                          {(bangMe?.hand ?? []).map((card) => (
                            <button
                              className={`bang-card ${card.type} ${bangDiscardSelectedIds.includes(card.id) ? "selected" : ""}`}
                              key={card.id}
                              onClick={() => {
                                if (bangMustDiscard) {
                                  setSelectedBangDiscardIds((previous) =>
                                    toggleLimitedSelected(previous, card.id, bangPendingDiscardCount)
                                  );
                                  return;
                                }
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "bang.play",
                                  setCurrentSession,
                                  setRoomMessage,
                                  bangPlayPayload(card, bangSelectedTargetId)
                                );
                              }}
                              disabled={
                                bangMustDiscard
                                  ? false
                                  : !bangCardPlayable(
                                      card,
                                      isMyTurn,
                                      bangMe,
                                      bangSelectedTargetId,
                                      bangAliveCount,
                                      bangActionBlocked
                                    )
                              }
                            >
                              <BangCardFace card={card} />
                            </button>
                          ))}
                        </div>
                        <div className="bang-target-panel" aria-label="공격 대상">
                          <strong>공격 대상</strong>
                          <div className="bang-targets">
                            {bangTargets.map((player) => (
                              <button
                                className={`bang-target-chip ${bangSelectedTargetId === player.playerId ? "selected" : ""}`}
                                key={player.playerId}
                                onClick={() => setSelectedBangTargetId(player.playerId)}
                                disabled={!isMyTurn || bangActionBlocked}
                              >
                                {participantName(currentRoom, player.playerId)}
                                <span>
                                  HP {player.hp ?? 0}/{player.maxHp ?? 0}
                                </span>
                              </button>
                            ))}
                          </div>
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
                          disabled={!isMyTurn || Boolean(bangMe?.drawn) || bangActionBlocked}
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
                          disabled={!isMyTurn || !bangMe?.drawn || bangActionBlocked}
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
                              <HwatuCardFace card={card} />
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
                              <HwatuCardFace card={card} />
                            </button>
                          ))}
                        </div>
                        <div className="hwatu-field">
                          {(currentSession.state.field ?? []).map((card) => (
                            <span className={`hwatu-card ${card.kind}`} key={card.id}>
                              <HwatuCardFace card={card} />
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
                          disabled={!isMyTurn || !currentSession.state.awaitingDecision || (gostopMe?.score ?? 0) < 3}
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
                          disabled={!isMyTurn || !currentSession.state.awaitingDecision || (gostopMe?.score ?? 0) < 3}
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
                            더미 {currentSession.state.drawPile?.length ?? 0}장 · 내 손패 {onecardMe?.hand?.length ?? 0}장 · 공격 누적{" "}
                            {currentSession.state.pendingDraw ?? 0}장
                            {currentSession.state.declaredSuit ? ` · 선언 ${standardSuitLabel(currentSession.state.declaredSuit)}` : ""}
                          </span>
                        </div>
                        <div className="rule-message-list">
                          <span>{onecardRuleSummary(currentSession.state.rules)}</span>
                          {currentSession.state.ruleMessages?.map((message) => (
                            <span key={message}>{message}</span>
                          ))}
                        </div>
                        <div className="option-grid compact-option-grid">
                          <label>
                            선언 문양
                            <select value={onecardDeclaredSuit} onChange={(event) => setOnecardDeclaredSuit(event.target.value)}>
                              <option value="spade">스페이드</option>
                              <option value="heart">하트</option>
                              <option value="diamond">다이아</option>
                              <option value="club">클럽</option>
                            </select>
                          </label>
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
                                  onecardPlayPayload(card, currentSession.state.rules, onecardDeclaredSuit, onecardMe?.hand?.length ?? 0)
                                )
                              }
                              disabled={!isMyTurn}
                            >
                              <PlayingCardFace card={card} />
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
                        <button
                          className="wide-button"
                          onClick={() =>
                            sendGameAction(
                              currentSession.id,
                              currentRoom?.id,
                              "onecard.declare_one",
                              setCurrentSession,
                              setRoomMessage
                            )
                          }
                          disabled={
                            !onecardMe ||
                            (onecardMe.hand?.length ?? onecardMe.handSize ?? 0) !== 1 ||
                            currentSession.state.declaredOne?.[playerID] === true ||
                            currentSession.state.rules?.oneCardPenalty === false
                          }
                        >
                          원카드 선언
                          <ChevronRight size={18} />
                        </button>
                        {(currentSession.state.players ?? [])
                          .filter(
                            (player) =>
                              player.playerId !== playerID &&
                              (player.handSize ?? player.hand?.length ?? 0) === 1 &&
                              currentSession.state.declaredOne?.[player.playerId] !== true &&
                              currentSession.state.rules?.oneCardPenalty !== false
                          )
                          .map((player) => (
                            <button
                              className="wide-button dark"
                              key={`callout-${player.playerId}`}
                              onClick={() =>
                                sendGameAction(
                                  currentSession.id,
                                  currentRoom?.id,
                                  "onecard.callout_one",
                                  setCurrentSession,
                                  setRoomMessage,
                                  { targetPlayerId: player.playerId }
                                )
                              }
                            >
                              {participantName(currentRoom, player.playerId)} 미선언 지적
                              <ChevronRight size={18} />
                            </button>
                          ))}
                        {currentSession.state.finished ? (
                          <p className="helper-copy">승자 {participantName(currentRoom, currentSession.state.winnerId ?? "")}</p>
                        ) : null}
                      </div>
                    ) : currentSession.gameId === "jokerdraw" ? (
                      <div className="room-actions">
                        <div className="standard-hand" aria-label="내 조커뽑기 손패">
                          {(jokerdrawMe?.hand ?? []).map((card) => (
                            <span className={`standard-card readonly ${card.suit ?? ""}`} key={card.id}>
                              <PlayingCardFace card={card} />
                            </span>
                          ))}
                        </div>
                        <div className="joker-targets">
                          {jokerdrawTargets.length === 0 ? (
                            <div className="empty-party-state">
                              <strong>{isMyTurn ? "뽑을 대상이 없습니다." : `${currentTurnName} 님 차례입니다.`}</strong>
                              <small>차례가 오면 다음 플레이어의 뒷면 카드가 표시됩니다.</small>
                            </div>
                          ) : null}
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
                                    <PlayingCardBack />
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
                            {werewolfMe?.originalRole === "werewolf" ? (
                              <>
                                <button
                                  className="wide-button"
                                  onClick={() =>
                                    sendGameAction(
                                      currentSession.id,
                                      currentRoom?.id,
                                      "werewolf.see_werewolves",
                                      setCurrentSession,
                                      setRoomMessage
                                    )
                                  }
                                >
                                  늑대 동료 확인
                                  <ChevronRight size={18} />
                                </button>
                                {[0, 1, 2].map((centerIndex) => (
                                  <button
                                    className="wide-button dark"
                                    key={`lone-wolf-center-${centerIndex}`}
                                    onClick={() =>
                                      sendGameAction(
                                        currentSession.id,
                                        currentRoom?.id,
                                        "werewolf.lone_wolf_center",
                                        setCurrentSession,
                                        setRoomMessage,
                                        { centerIndexes: [centerIndex] }
                                      )
                                    }
                                  >
                                    외로운 늑대 중앙 {centerIndex + 1}번
                                    <ChevronRight size={18} />
                                  </button>
                                ))}
                              </>
                            ) : null}
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
                            {davinciTileLabel(tile)}
                          </span>
                        ))}
                      </div>
                    </div>
                    {currentSession.state.pendingTile ? (
                      <div className="tile-board pending-davinci-tile" aria-label="이번 턴에 뽑은 타일">
                        <span>
                          {currentSession.state.pendingOwnerId === playerID
                            ? "이번 턴에 뽑은 타일"
                            : `${participantName(currentRoom, currentSession.state.pendingOwnerId ?? "")} 님이 뽑은 타일`}
                        </span>
                        <div className="tile-row">
                          <span className={`davinci-tile ${currentSession.state.pendingTile.color}`}>
                            {davinciTileLabel(currentSession.state.pendingTile)}
                          </span>
                        </div>
                        {currentSession.state.pendingOwnerId === playerID && currentSession.state.pendingTile.joker ? (
                          <label className="joker-insert-control">
                            삽입 위치
                            <input
                              type="number"
                              min="0"
                              max={myDavinciPlayer?.tiles?.length ?? 0}
                              value={pendingInsertIndex}
                              onChange={(event) => setPendingInsertIndex(Number(event.target.value))}
                            />
                          </label>
                        ) : null}
                      </div>
                    ) : null}
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
                                {davinciTileLabel(tile)}
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
                        disabled={guessJoker}
                        aria-label="추측 숫자"
                      />
                      <label className="checkbox-inline">
                        <input
                          type="checkbox"
                          checked={guessJoker}
                          onChange={(event) => setGuessJoker(event.target.checked)}
                        />
                        조커
                      </label>
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
                            value: guessValue,
                            joker: guessJoker,
                            insertIndex: pendingInsertIndex
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
                          setRoomMessage,
                          { insertIndex: pendingInsertIndex }
                        )
                      }
                      disabled={!isMyTurn || !currentSession.state.canEndTurn}
                    >
                      턴 종료
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
            <div className="room-card chat-card">
              <div className="section-title">
                <div>
                  <span>{currentSession ? "인게임" : "채팅"}</span>
                  <h2>{currentSession ? "채팅/감정표현" : "로비 메시지"}</h2>
                </div>
                <MessageCircle size={22} />
              </div>
              <div className="chat-list">
                {tail(chatMessages, 5).map((message) => (
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
                      setChatMessages((previous) => appendChatMessage(previous, message));
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
                      setChatMessages((previous) => appendChatMessage(previous, message));
                      setChatInput("");
                    });
                  }}
                >
                  전송
                </button>
              </div>
              <div className="emoji-row">
                {["👍", "🎉", "😮", "👏", "🔥"].map((emoji) => (
                  <button
                    key={emoji}
                    onClick={() =>
                      sendChat(currentRoom.id, emoji, "emoji").then((message) =>
                        setChatMessages((previous) => appendChatMessage(previous, message))
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
                {authSession ? (
                  <small className="status-badge">{authSession.user.role === "ADMIN" ? "ADMIN" : "USER"}</small>
                ) : null}
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
                  setLobbyFlow("home");
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
              <h2>
                {currentRoom
                  ? `${activePlayerCount}명이 바로 플레이 가능한 게임`
                  : `${recommendationPlayerCount}명 기준 추천 게임`}
              </h2>
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

          {!currentRoom && (lobbyFlow === "public-room" || lobbyFlow === "game-rooms") ? (
            <div className="public-room-panel">
              <div className="section-title">
                <div>
                  <span>Public Rooms</span>
                  <h2>{lobbyFlow === "game-rooms" ? `${selectedGame.title} 공개 방` : "입장 가능한 공개 방"}</h2>
                </div>
                <button
                  className="mini-command"
                  onClick={() =>
                    fetchPublicRooms(lobbyFlow === "game-rooms" ? selectedGameId : "")
                      .then(setPublicRooms)
                      .catch(() => setPublicRooms([]))
                  }
                >
                  새로고침
                </button>
              </div>
              <div className="public-room-list">
                {publicRooms.length === 0 ? (
                  <div className="empty-party-state">
                    <strong>아직 공개 방이 없습니다.</strong>
                    <small>공개 방을 만들거나 퀵매칭으로 새 방을 열 수 있습니다.</small>
                  </div>
                ) : (
                  publicRooms.map((room) => {
                    const roomGame = games.find((game) => game.id === room.gameId);
                    return (
                      <article className="public-room-item" key={room.id}>
                        <div>
                          <strong>{roomGame?.title ?? room.gameId}</strong>
                          <small>
                            코드 {room.code} · {room.status} · 참가 {room.participants.length}/{room.maxPlayers} · 관전{" "}
                            {room.spectators?.length ?? 0}
                          </small>
                        </div>
                        <button onClick={() => joinPublicRoom(room.id, setCurrentRoom, setRoomMessage)}>
                          {room.status === "PLAYING" ? "관전 입장" : room.participants.length >= room.maxPlayers ? "관전 입장" : "입장"}
                        </button>
                      </article>
                    );
                  })
                )}
              </div>
            </div>
          ) : null}

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
                head(leaderboard, 5).map((row, index) => (
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
                setLobbyFlow("home");
                setRoomMessage("로컬 방/게임 복구 정보를 초기화했습니다.");
              }}
            >
              로컬 세션 초기화
            </button>
          </div>
        </section>
      </section>
      ) : null}

      <section className="dock" aria-label="모바일 빠른 실행">
        <button onClick={() => setLobbyFlow("home")}>
          <Gamepad2 size={20} />
          로비
        </button>
        <button onClick={() => setLobbyFlow("public-room")}>
          <Bot size={20} />
          공개
        </button>
        <button className="dock-primary" onClick={() => setLobbyFlow("quick-match")}>
          <Play size={20} />
          시작
        </button>
        <button onClick={() => setLobbyFlow("private-room")}>
          <Crown size={20} />
          비공개
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

function participantName(room: Room | null, playerID: string) {
  return room?.participants.find((participant) => participant.user.id === playerID)?.user.nickname ?? "상대";
}

function nextActivePlayerAfter(players: DavinciPlayer[], currentIndex: number) {
  if (players.length === 0) return undefined;
  for (let step = 1; step <= players.length; step++) {
    const player = players[(currentIndex + step) % players.length];
    if (player?.active && !player.out) return player;
  }
  return undefined;
}

function davinciTileLabel(tile: DavinciTile) {
  if (tile.color === "hidden") return "?";
  if (tile.joker) return "-";
  return tile.value >= 0 ? String(tile.value) : "?";
}

function onecardRuleSummary(rules: OneCardRules | undefined) {
  if (!rules) return "원카드 기본 룰";
  return `공격 ${onecardAttackLabel(rules.attackCards)} · 방어 ${onecardDefenseLabel(
    rules.defenseMode
  )} · 조커 ${rules.jokerDrawCount ?? 5}장 · 누적 ${rules.stacking === false ? "없음" : "허용"} · 문양 ${onecardChangeSuitLabel(
    rules.changeSuitCards
  )} · 원카드 ${rules.oneCardPenalty === false ? "벌칙 없음" : `${rules.oneCardPenaltyDraw ?? 2}장 벌칙`} · 막카 공격 ${
    rules.allowFinalAttack === false ? "금지" : "허용"
  } · 막카 특수 ${rules.allowFinalSpecial === false ? "금지" : "허용"}`;
}

function onecardPlayPayload(card: HandCard, rules: OneCardRules | undefined, declaredSuit: string, handSize: number) {
  const payload: Record<string, unknown> = { cardId: card.id };
  if (onecardCanChangeSuit(card, rules)) {
    payload.declaredSuit = declaredSuit;
  }
  if (rules?.oneCardPenalty !== false && handSize === 2) {
    payload.declareOne = true;
  }
  return payload;
}

function onecardCanChangeSuit(card: HandCard, rules: OneCardRules | undefined) {
  const changeCards = rules?.changeSuitCards ?? ["7", "JOKER"];
  if (changeCards.length === 0) return false;
  if (card.joker) return changeCards.includes("JOKER");
  return changeCards.includes(String(card.rank ?? ""));
}

function onecardAttackLabel(cards: string[] | undefined) {
  const value = (cards ?? []).join("/");
  if (value === "2") return "2만";
  if (value === "2/A") return "2/A";
  return "2/A/조커";
}

function onecardDefenseLabel(mode: string | undefined) {
  const labels: Record<string, string> = {
    "same-rank": "같은 공격카드만",
    "any-attack": "공격카드",
    "attack-or-joker": "공격카드/조커"
  };
  return labels[mode ?? "attack-or-joker"] ?? "공격카드/조커";
}

function onecardChangeSuitLabel(cards: string[] | undefined) {
  const value = (cards ?? ["7", "JOKER"]).join("/");
  if (value === "") return "없음";
  if (value === "7") return "7";
  if (value === "JOKER") return "조커";
  if (value === "Q/JOKER") return "Q/조커";
  return "7/조커";
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
    black: "검정",
    gold: "금"
  };
  return labels[color] ?? color;
}

function romanTier(tier: number) {
  const labels: Record<number, string> = {
    1: "I",
    2: "II",
    3: "III"
  };
  return labels[tier] ?? String(tier);
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

function rummikubSubmitPayload(groups: string[][]) {
  if (groups.length === 1) {
    return { tileIds: groups[0] };
  }
  return { groups };
}

function rummikubTileLabel(rack: RummikubTile[], id: string) {
  const tile = rack.find((item) => item.id === id);
  if (!tile) return id;
  return tile.joker ? "J" : `${tile.color} ${tile.number}`;
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

function bangCardLabel(cardType: string) {
  const labels: Record<string, string> = {
    bang: "BANG!",
    gatling: "개틀링",
    missed: "빗맞음",
    beer: "맥주"
  };
  return labels[cardType] ?? cardType;
}

function bangPlayPayload(card: HandCard, targetPlayerId: string) {
  const payload: Record<string, string> = { cardId: card.id };
  if (card.type === "bang") {
    payload.targetPlayerId = targetPlayerId;
  }
  return payload;
}

function bangCardPlayable(
  card: HandCard,
  isMyTurn: boolean,
  player: DavinciPlayer | undefined,
  targetPlayerId: string,
  aliveCount: number,
  hasPendingAttack: boolean
) {
  if (hasPendingAttack) return false;
  if (!isMyTurn || !player?.drawn) return false;
  if (card.type === "bang") return Boolean(targetPlayerId) && !player.bangUsed;
  if (card.type === "gatling") return true;
  if (card.type === "beer") return (player.hp ?? 0) < (player.maxHp ?? 0) && aliveCount > 2;
  return false;
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

function toggleLimitedSelected(values: string[], target: string, limit: number) {
  if (values.includes(target)) {
    return values.filter((value) => value !== target);
  }
  if (values.length >= limit) {
    return values;
  }
  return [...values, target];
}

function toggleGemColor(selected: string[], color: string): string[] {
  if (selected.length === 0) {
    return [color];
  }
  if (selected.length === 1) {
    return selected[0] === color ? [color, color] : [...selected, color];
  }
  if (selected.length === 2 && selected[0] !== selected[1] && !selected.includes(color)) {
    return [...selected, color];
  }
  return selected;
}

function toggleReturnGemColor(selected: string[], color: string, limit: number, available: number): string[] {
  const selectedCount = selected.filter((item) => item === color).length;
  if (selectedCount >= available || selected.length >= limit) {
    const index = selected.indexOf(color);
    if (index < 0) return selected;
    return [...selected.slice(0, index), ...selected.slice(index + 1)];
  }
  return [...selected, color];
}

function countSelectedColors(selected: string[]) {
  return selected.reduce<Record<string, number>>((counts, color) => {
    counts[color] = (counts[color] ?? 0) + 1;
    return counts;
  }, {});
}

function isValidGemSelection(selected: string[]): boolean {
  if (selected.length === 3) {
    return new Set(selected).size === 3;
  }
  if (selected.length === 2) {
    return selected[0] === selected[1];
  }
  return false;
}

function isValidReturnGemSelection(selected: string[], requiredCount: number): boolean {
  return requiredCount > 0 && selected.length === requiredCount;
}
