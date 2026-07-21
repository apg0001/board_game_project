import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";

interface FlowOptionCardProps {
  icon: ReactNode;
  title: string;
  description: string;
  primary?: boolean;
  onClick: () => void;
}

export function FlowOptionCard({ icon, title, description, primary = false, onClick }: FlowOptionCardProps) {
  return (
    <button className={`flow-card ${primary ? "primary-flow" : ""}`} onClick={onClick}>
      <span>{icon}</span>
      <strong>{title}</strong>
      <small>{description}</small>
      <ChevronRight size={20} />
    </button>
  );
}
