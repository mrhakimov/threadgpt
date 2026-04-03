"use client"

import React from "react"
import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import ChatInput from "./ChatInput"

describe("ChatInput", () => {
  it("submits trimmed content on Enter and clears the field", () => {
    const onSend = vi.fn()

    render(<ChatInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("Message ThreadGPT")
    fireEvent.change(input, { target: { value: "  hello world  " } })
    fireEvent.keyDown(input, { key: "Enter" })

    expect(onSend).toHaveBeenCalledWith("hello world")
    expect(input).toHaveValue("")
  })

  it("keeps multiline editing when Shift+Enter is used", () => {
    const onSend = vi.fn()

    render(<ChatInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("Message ThreadGPT")
    fireEvent.change(input, { target: { value: "line one" } })
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true })

    expect(onSend).not.toHaveBeenCalled()
    expect(input).toHaveValue("line one")
  })
})
