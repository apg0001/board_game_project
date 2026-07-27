import type { HandCard, SplendorCard, SplendorNoble } from "../domain/types";
import { playingCardAsset, playingCardBackAsset } from "../domain/cardAssets";

const suitSymbol: Record<string, string> = {
  spade: "♠",
  heart: "♥",
  diamond: "◆",
  club: "♣",
  joker: "★"
};

const suitLabel: Record<string, string> = {
  spade: "스페이드",
  heart: "하트",
  diamond: "다이아",
  club: "클럽",
  joker: "조커"
};

const hwatuKindLabel: Record<string, string> = {
  bright: "광",
  animal: "열끗",
  ribbon: "띠",
  junk: "피"
};

const bangLabel: Record<string, string> = {
  bang: "BANG!",
  missed: "빗나감",
  beer: "맥주",
  gatling: "개틀링"
};

const gemLabel: Record<string, string> = {
  white: "흰색",
  blue: "파랑",
  green: "초록",
  red: "빨강",
  black: "검정",
  gold: "금"
};

interface CardFaceProps {
  card: HandCard;
  hidden?: boolean;
}

interface DalmutiCardFanProps {
  rank: number;
  count: number;
}

interface SplendorCardFaceProps {
  card: SplendorCard;
  badge?: string;
}

interface SplendorNobleFaceProps {
  noble: SplendorNoble;
}

export function PlayingCardFace({ card, hidden = false }: CardFaceProps) {
  if (hidden) return <PlayingCardBack />;

  const asset = playingCardAsset(card);
  if (asset) {
    return (
      <span className={`card-face playing-card-face asset-card suit-${card.suit ?? (card.joker ? "joker" : "")}`}>
        <img src={asset} alt={playingCardAlt(card)} loading="lazy" />
      </span>
    );
  }

  const suit = card.suit ?? (card.joker ? "joker" : "");
  const rank = card.joker ? "JOKER" : String(card.rank ?? "?");

  return (
    <span className={`card-face playing-card-face suit-${suit}`}>
      <span className="card-corner top">
        <strong>{rank}</strong>
        <small>{suitSymbol[suit] ?? ""}</small>
      </span>
      <span className="card-center-mark">{suitSymbol[suit] ?? rank}</span>
      <span className="card-title">{card.joker ? "JOKER" : suitLabel[suit] ?? suit}</span>
      <span className="card-corner bottom">
        <strong>{rank}</strong>
        <small>{suitSymbol[suit] ?? ""}</small>
      </span>
    </span>
  );
}

export function PlayingCardBack() {
  return (
    <span className="card-face playing-card-back">
      <img src={playingCardBackAsset} alt="카드 뒷면" loading="lazy" />
    </span>
  );
}

export function HwatuCardFace({ card }: CardFaceProps) {
  const month = card.month ?? "?";
  const kind = card.kind ?? "junk";
  return (
    <span className={`card-face hwatu-card-face kind-${kind}`}>
      <span className="hwatu-month">{month}월</span>
      <span className="hwatu-branch" />
      <span className="hwatu-mark">{card.gwang ? "光" : hwatuKindLabel[kind] ?? kind}</span>
    </span>
  );
}

export function BangCardFace({ card }: CardFaceProps) {
  const type = card.type ?? "";
  return (
    <span className={`card-face bang-card-face type-${type}`}>
      <span className="bang-card-kicker">BANG!</span>
      <strong>{bangLabel[type] ?? type}</strong>
      <small>{type === "bang" ? "공격" : type === "missed" ? "반응" : "액션"}</small>
    </span>
  );
}

export function DalmutiCardFan({ rank, count }: DalmutiCardFanProps) {
  const label = dalmutiRankLabel(rank);
  const visible = Math.min(count, 4);
  return (
    <span className="dalmuti-card-fan" aria-label={`${label} ${count}장`}>
      {Array.from({ length: visible }, (_, index) => (
        <span className={`card-face dalmuti-card-face rank-${rank}`} key={`${rank}-${index}`}>
          <small>{rank === 13 ? "J" : rank}</small>
          <strong>{label}</strong>
        </span>
      ))}
      {count > visible ? <em>+{count - visible}</em> : null}
    </span>
  );
}

export function SplendorCardFace({ card, badge }: SplendorCardFaceProps) {
  return (
    <span className={`card-face splendor-card-face gem-${card.color}`}>
      <span className="splendor-points">{card.points}점</span>
      <strong>{gemLabel[card.color] ?? card.color}</strong>
      <small>{formatCost(card.cost)}</small>
      {badge ? <span className="splendor-badge">{badge}</span> : null}
    </span>
  );
}

export function SplendorNobleFace({ noble }: SplendorNobleFaceProps) {
  return (
    <span className="card-face splendor-noble-face">
      <span className="splendor-points">{noble.points}점</span>
      <strong>귀족</strong>
      <small>{formatCost(noble.cost)}</small>
    </span>
  );
}

function dalmutiRankLabel(rank: number) {
  if (rank === 1) return "달무티";
  if (rank === 13) return "광대";
  return `${rank}계급`;
}

function formatCost(cost: Record<string, number>) {
  const parts = Object.entries(cost)
    .filter(([, value]) => value > 0)
    .map(([color, value]) => `${gemLabel[color] ?? color} ${value}`);
  return parts.length > 0 ? parts.join(" · ") : "무료";
}

function playingCardAlt(card: HandCard) {
  if (card.joker || card.rank === "JOKER") return "조커";
  const suit = card.suit ? suitLabel[card.suit] ?? card.suit : "카드";
  return `${suit} ${card.rank ?? ""}`.trim();
}
