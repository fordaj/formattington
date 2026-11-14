extends Node

func _ready() -> void:
	print("hello")

func _input(event: InputEvent) -> void:
	if event is InputEventKey:
		pass
