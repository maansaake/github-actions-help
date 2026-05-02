import yaml

def main():
  print("Hello, World!")

  with open("output.yaml", "w") as f:
    yaml.dump({"message": "Hello, World!"}, f)

if __name__ == "__main__":
  main()
