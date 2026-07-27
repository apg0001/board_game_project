export type GameCategory = "블러핑" | "전략" | "순발력" | "숫자/조합" | "파티" | "카드";

export interface GameCard {
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

export interface ApiGame {
  id: string;
  title: string;
  minPlayers: number;
  maxPlayers: number;
  estimatedMinutes: number;
  difficulty: "쉬움" | "보통" | "어려움";
  categories: GameCategory[];
}

export interface GuestSession {
  sessionToken: string;
  user: {
    id: string;
    nickname: string;
  };
}

export interface AuthSession {
  sessionToken: string;
  user: {
    id: string;
    username: string;
    nickname: string;
    role: "USER" | "ADMIN";
  };
}

export interface RoomParticipant {
  user: {
    id: string;
    nickname: string;
  };
  ready: boolean;
  host: boolean;
  seatIndex: number;
}

export interface RoomSpectator {
  user: {
    id: string;
    nickname: string;
  };
}

export interface RuleVote {
  userId: string;
  choices: Record<string, string>;
}

export interface Room {
  id: string;
  code: string;
  gameId: string;
  visibility: "PRIVATE" | "PUBLIC";
  status: "LOBBY" | "PLAYING" | "FINISHED" | "CLOSED";
  activeSessionId?: string;
  playingPlayerIds?: string[];
  maxPlayers: number;
  spectators: RoomSpectator[];
  options: {
    turnSeconds: number;
    maxWaitSeconds: number;
    autoStart: boolean;
    allowSpectators: boolean;
  };
  participants: RoomParticipant[];
  ruleVotes?: RuleVote[];
  gameRules?: Record<string, unknown>;
  ruleMessages?: string[];
}

export type RoomOptions = Room["options"];

export type RoomSettings = RoomOptions & {
  maxPlayers: number;
};

export interface DavinciTile {
  color: "black" | "white" | "hidden";
  value: number;
  joker?: boolean;
  revealed: boolean;
}

export interface DavinciPlayer {
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
  reserved?: SplendorCard[];
  nobles?: SplendorNoble[];
  hand?: HandCard[];
  equipment?: HandCard[];
  handSize?: number;
  characterId?: string;
  characterName?: string;
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

export interface HandCard {
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

export interface RummikubTile {
  id: string;
  color: string;
  number: number;
  joker: boolean;
}

export interface SplendorCard {
  id: string;
  tier?: number;
  color: string;
  points: number;
  cost: Record<string, number>;
  hidden?: boolean;
}

export interface SplendorNoble {
  id: string;
  points: number;
  cost: Record<string, number>;
}

export interface OneCardRules {
  attackCards?: string[];
  defenseMode?: string;
  jokerDrawCount?: number;
  twoDrawCount?: number;
  stacking?: boolean;
  changeSuitCards?: string[];
  oneCardPenalty?: boolean;
  oneCardPenaltyDraw?: number;
  allowFinalAttack?: boolean;
  allowFinalSpecial?: boolean;
}

export interface GameSession {
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
    markets?: Record<string, SplendorCard[]>;
    decks?: Record<string, SplendorCard[]>;
    nobles?: SplendorNoble[];
    pendingReturnPlayerId?: string;
    pendingReturnCount?: number;
    pendingNoblePlayerId?: string;
    pendingNobleChoices?: SplendorNoble[];
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
    awaitingDecision?: boolean;
    loserId?: string;
    field?: HandCard[];
    discardPile?: HandCard[];
    drawPile?: HandCard[];
    pendingAttack?: {
      sourcePlayerId: string;
      targetPlayerId: string;
      cardType: string;
      damage: number;
      requiredResponseCount?: number;
      remainingTargetIds?: string[];
    };
    pendingGeneralStore?: {
      offer: HandCard[];
      currentChooserId: string;
      remainingChooserIds?: string[];
    };
    pendingCharacterChoice?: {
      playerId: string;
      characterId: string;
      choiceType: string;
      cards: HandCard[];
      requiredCount?: number;
    };
    pendingDiscardPlayerId?: string;
    pendingDiscardCount?: number;
    deck?: DavinciTile[];
    pendingTile?: DavinciTile;
    pendingOwnerId?: string;
    canEndTurn?: boolean;
    rules?: OneCardRules;
    ruleMessages?: string[];
    pendingDraw?: number;
    pendingAttackRank?: string;
    declaredSuit?: string;
    declaredOne?: Record<string, boolean>;
  };
  results?: Array<{
    playerId: string;
    rank: number;
    score: number;
    outcome: "WIN" | "LOSE" | "DRAW";
  }>;
}

export interface RealtimeMessage {
  room: string;
  type: string;
  payload: {
    room?: Room;
    session?: GameSession;
    message?: ChatMessage;
    presence?: Presence;
  };
}

export interface ChatMessage {
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

export interface Presence {
  userId: string;
  roomId: string;
  sessionId?: string;
  status: "ONLINE" | "DISCONNECTED";
  expiresAt?: string;
}

export interface LeaderboardRow {
  userId: string;
  gameId: string;
  wins: number;
  losses: number;
  draws: number;
  playCount: number;
  mmr: number;
}

export interface TutorialGuide {
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

export type LobbyFlow = "home" | "private-room" | "public-room" | "quick-match" | "game-rooms" | "room";

export type GameActionType =
  | "davinci.pass"
  | "davinci.finish"
  | "halli-galli.flip"
  | "halli-galli.ring"
  | "splendor.take_token"
  | "splendor.buy_card"
  | "splendor.reserve_card"
  | "splendor.return_tokens"
  | "splendor.choose_noble"
  | "dalmuti.play"
  | "dalmuti.pass"
  | "werewolf.see_werewolves"
  | "werewolf.lone_wolf_center"
  | "werewolf.see_player"
  | "werewolf.see_center"
  | "werewolf.rob"
  | "werewolf.troublemake"
  | "werewolf.drunk_swap"
  | "werewolf.finish_night"
  | "werewolf.vote"
  | "rummikub.meld"
  | "rummikub.draw"
  | "rummikub.rearrange"
  | "bang.draw"
  | "bang.play"
  | "bang.use_bang"
  | "bang.use_missed"
  | "bang.take_hit"
  | "bang.choose_general_store"
  | "bang.choose_character_card"
  | "bang.sid_heal"
  | "bang.discard"
  | "bang.end_turn"
  | "sutda.call"
  | "sutda.fold"
  | "sutda.showdown"
  | "gostop.play"
  | "gostop.go"
  | "gostop.stop"
  | "onecard.play"
  | "onecard.draw"
  | "onecard.declare_one"
  | "onecard.callout_one"
  | "jokerdraw.draw";
